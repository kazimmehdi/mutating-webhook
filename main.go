package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type WebhookConfig struct {
	Labels map[string]string `json:"labels"`
	TopologySpreadConstraints []interface{} `json:""`
}

var config WebhookConfig

func main() {
	loadConfig()
	http.HandleFunc("/mutate", handleMutate)
	http.HandleFunc("/health", handleHealth)
	log.Printf("Starting webhook server on :8443 with labels: %v, topologySpreadConstrainst: %v", config.Labels, config.TopologySpreadConstraints)
	if err := http.ListenAndServeTLS(":8443", "/etc/webhook/certs/tls.crt", "/etc/webhook/certs/tls.key", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func loadConfig() {
	// Loading Labels Config file
	labelsConfigFile := os.Getenv("LABELS_CONFIG_FILE")
	if labelsConfigFile == "" {
		labelsConfigFile = "/etc/webhook/config/labels.json"
	}
	data, err := os.ReadFile(labelsConfigFile)
	if err != nil {
		log.Printf("Warning: Could not read config file %s: %v", labelsConfigFile, err)
		return
	}
	if err := json.Unmarshal(data, &config.Labels); err != nil {
		log.Printf("Warning: Could not parse config file: %v", err)
		return
	}
	log.Printf("Loaded configuration from file: %s", labelsConfigFile)
	if len(config.Labels) == 0 {
		config.Labels = map[string]string{"mutated": "true"}
		log.Println("No labels configured, using default: mutated=true")
	}

	//Loading TopologySpreadConstraints Config File
	topologySpreadConstraintConfigFile := os.Getenv("TOPOLOGY_CONFIG_FILE")
	if topologySpreadConstraintConfigFile == "" {
		topologySpreadConstraintConfigFile = "/etc/webhook/config/topologySpreadConstraints.json"
	}
	topologyData, err := os.ReadFile(topologySpreadConstraintConfigFile)
	if err != nil {
		log.Printf("Warning: Could not read config file %s: %v", topologySpreadConstraintConfigFile, err)
		return
	}
	if err := json.Unmarshal(topologyData, &config.TopologySpreadConstraints); err != nil {
		log.Printf("Warning: Could not parse config file: %v", err)
		return
	}
	log.Printf("Loaded configuration from file: %s", topologySpreadConstraintConfigFile)
	if len(config.TopologySpreadConstraints) == 0 {
		log.Println("No topology spread constraint configured")
	}

}


func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func handleMutate(w http.ResponseWriter, r *http.Request) {
	log.Println("Received mutation request")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	admissionReview := admissionv1.AdmissionReview{}
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		log.Printf("Failed to unmarshal admission review: %v", err)
		http.Error(w, "Failed to parse admission review", http.StatusBadRequest)
		return
	}
	admissionResponse := &admissionv1.AdmissionResponse{
		UID:     admissionReview.Request.UID,
		Allowed: true,
	}
	pod := corev1.Pod{}
	if err := json.Unmarshal(admissionReview.Request.Object.Raw, &pod); err != nil {
		log.Printf("Failed to unmarshal pod: %v", err)
		admissionResponse.Result = &metav1.Status{
			Message: fmt.Sprintf("Failed to unmarshal pod: %v", err),
		}
	} else {
		patch := createPatch(&pod)
		if len(patch) > 0 {
			patchBytes, err := json.Marshal(patch)
			if err != nil {
				log.Printf("Failed to marshal patch: %v", err)
			} else {
				admissionResponse.Patch = patchBytes
				admissionResponse.PatchType = new(admissionv1.PatchType)
				*admissionResponse.PatchType = admissionv1.PatchTypeJSONPatch
				log.Printf("Applied patch to pod %s/%s with labels: %v", pod.Namespace, pod.Name, config.Labels)
			}
		}
	}
	responseAdmissionReview := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: admissionResponse,
	}
	responseBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		http.Error(w, "Failed to create response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBytes)
}

func createPatch(pod *corev1.Pod) []patchOperation {
	
	labelsToAdd := config.Labels
	labels_patches, _ := patchOrReplaceLabels(pod, labelsToAdd)	

	topology_constraint := config.TopologySpreadConstraints
	topology_constraint_patches, _ := patchOrReplaceTopologySpreadConstraints(pod, topology_constraint)
    
	patches := append(labels_patches, topology_constraint_patches...)

	log.Println("Printing patches array")
	fmt.Println(patches)

	return patches
}


func patchOrReplaceLabels(pod *corev1.Pod, labelsToAdd map[string]string) ([]patchOperation, error) {
	var patches []patchOperation
	if pod == nil {
		return nil, fmt.Errorf("pod cannot be nil")
	}

	if pod.Labels == nil || len(pod.Labels) == 0 {
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/metadata/labels",
			Value: labelsToAdd,
		})
	} else {
		for key, value := range labelsToAdd {
			if _, exists := pod.Labels[key]; !exists {
				patches = append(patches, patchOperation{
					Op:    "add",
					Path:  fmt.Sprintf("/metadata/labels/%s", escapeJSONPointer(key)),
					Value: value,
				})
			} else {
				log.Printf("Label %s already exists on pod, skipping", key)
			}
		}
	}
	log.Printf("Created label patch with %d operations", len(patches))

	return patches, nil
}

func patchOrReplaceTopologySpreadConstraints(pod *corev1.Pod, newConstraints []interface{}) ([]patchOperation, error) {
	var patches []patchOperation
	if pod == nil {
		return nil, fmt.Errorf("pod cannot be nil")
	}

	// Check if newConstraints is nil or empty
	if newConstraints != nil && len(newConstraints) > 0 {
		log.Printf("new topology constraints with %d operations", len(newConstraints))
		// Check if topologySpreadConstraints exists
		if pod.Spec.TopologySpreadConstraints == nil || len(pod.Spec.TopologySpreadConstraints) == 0 {
			// Add operation - create new array
			patches = append(patches, patchOperation{
				Op:    "add",
				Path:  "/spec/topologySpreadConstraints",
				Value: newConstraints,
			})
			return patches, nil
		}

		// Create a map of existing constraints by topology key
		existingMap := make(map[string]int)
		for i, constraint := range pod.Spec.TopologySpreadConstraints {
			log.Printf("existing topology with key %s", constraint.TopologyKey)
			existingMap[constraint.TopologyKey] = i
		}

		// Process new constraints
		for _, newConstraint := range newConstraints {
			var newConstraintTopologyKey string = ""
			if m, ok := newConstraint.(map[string]interface{}); ok {
				newConstraintTopologyKey = m["topologyKey"].(string)
			}
			log.Printf("processing new topology with key %s", newConstraintTopologyKey)

			if idx, exists := existingMap[newConstraintTopologyKey]; exists {
				// Replace existing constraint at specific index
				patches = append(patches, patchOperation{
					Op:    "replace",
					Path:  fmt.Sprintf("/spec/topologySpreadConstraints/%d", idx),
					Value: newConstraint,
				})
			} else {
				// Add new constraint to end of array
				patches = append(patches, patchOperation{
					Op:    "add",
					Path:  "/spec/topologySpreadConstraints/-",
					Value: newConstraint,
				})
			}
		}
	}

	log.Printf("Created topology patch with %d operations", len(patches))

	return patches, nil
}

func escapeJSONPointer(s string) string {
	s = replaceAll(s, "~", "~0")
	s = replaceAll(s, "/", "~1")
	return s
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}
