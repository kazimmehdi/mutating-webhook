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
	NodeSelectorLabels map[string]string `json:"nodeSelectorLabels"`
	PodSelectorLabels map[string]string `json:"podSelectorLabels"`
}

var config WebhookConfig

func main() {
	loadConfig()
	http.HandleFunc("/mutate", handleMutate)
	http.HandleFunc("/health", handleHealth)
	log.Printf("Starting webhook server on :8443 with labels: %v, NodeSelectorLabels: %v , PodSelectorLabels: %v", config.Labels, config.NodeSelectorLabels, config.PodSelectorLabels)
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

	
	// Loading Pod Selector Lables Config file
	podSelectorLabelsConfigFile := os.Getenv("POD_SELECTOR_LABELS_CONFIG_FILE")
	if podSelectorLabelsConfigFile == "" {
		podSelectorLabelsConfigFile = "/etc/webhook/config/podSelectorLabels.json"
	}
	podSelectorLabelData, err := os.ReadFile(podSelectorLabelsConfigFile)
	if err != nil {
		log.Printf("Warning: Could not read config file %s: %v", podSelectorLabelsConfigFile, err)
		return
	}
	if err := json.Unmarshal(podSelectorLabelData, &config.PodSelectorLabels); err != nil {
		log.Printf("Warning: Could not parse config file: %v", err)
		return
	}
	log.Printf("Loaded configuration from file: %s", podSelectorLabelsConfigFile)


	// Loading Node Selector Lables Config file
	nodeSelectorLabelsConfigFile := os.Getenv("NODE_SELECTOR_LABELS_CONFIG_FILE")
	if nodeSelectorLabelsConfigFile == "" {
		nodeSelectorLabelsConfigFile = "/etc/webhook/config/nodeSelectorLabels.json"
	}
	nodeSelectorLabelData, err := os.ReadFile(nodeSelectorLabelsConfigFile)
	if err != nil {
		log.Printf("Warning: Could not read config file %s: %v", nodeSelectorLabelsConfigFile, err)
		return
	}
	if err := json.Unmarshal(nodeSelectorLabelData, &config.NodeSelectorLabels); err != nil {
		log.Printf("Warning: Could not parse config file: %v", err)
		return
	}
	log.Printf("Loaded configuration from file: %s", nodeSelectorLabelsConfigFile)
	
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

		if ok := containsAll(config.PodSelectorLabels, pod.Labels); ok {
			patch := createPatch(&pod)
		
			if len(patch) > 0 {
				patchBytes, err := json.Marshal(patch)
				if err != nil {
					log.Printf("Failed to marshal patch: %v", err)
				} else {
					admissionResponse.Patch = patchBytes
		
					patchType := admissionv1.PatchTypeJSONPatch
					admissionResponse.PatchType = &patchType
		
					log.Printf(
						"Applied patch to pod %s/%s with labels: %v",
						pod.Namespace,
						pod.Name,
						config.Labels,
					)
				}
			}
		} else {
			log.Printf("no pods with all the labels: %v found, therefore no mutation...", config.PodSelectorLabels)
			// ignore mutating but still return allowed
			admissionResponse = &admissionv1.AdmissionResponse{
				UID:     admissionReview.Request.UID,
				Allowed: true,
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

	nodeSelectorLabels := config.NodeSelectorLabels
	node_selector_labels_patches, _ := patchOrReplaceNodeSelectors(pod, nodeSelectorLabels)
    
	patches := append(labels_patches, node_selector_labels_patches...)

	log.Println("Printing patches array")
	fmt.Println(patches)

	return patches
}


func patchOrReplaceNodeSelectors(pod *corev1.Pod, nodeSelectorToAdd map[string]string) ([]patchOperation, error) {
	var patches []patchOperation
	if pod == nil {
		return nil, fmt.Errorf("pod cannot be nil")
	}

	// Ensure nodeSelector exists
	if pod.Spec.NodeSelector == nil {
		// Create the whole nodeSelector object
		patches = append(patches, patchOperation {
			Op:    "add",
			Path:  "/spec/nodeSelector",
			Value: nodeSelectorToAdd,
		})
	}

	// For each desired key, update or insert
	for k, v := range nodeSelectorToAdd {
		if _, exists := pod.Spec.NodeSelector[k]; exists {
			// Replace the value of an existing key
			patches = append(patches, patchOperation{
				Op:    "replace",
				Path:  "/spec/nodeSelector/" + escapeJSONPointer(k),
				Value: v,
			})
		} else {
			// Add new key
			patches = append(patches, patchOperation{
				Op:    "add",
				Path:  "/spec/nodeSelector/" + escapeJSONPointer(k),
				Value: v,
			})
		}
	}

	log.Printf("Created node selector labels patch with %d operations", len(patches))

	return patches, nil
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

/**
a -> label map provided by helm - looking to find pods containing all the labels from this map
b -> existing labels maps of the pod
**/
func containsAll(a, b map[string]string) bool {
	if b == nil || len(b) == 0 {
		return false
	}

    for k, v := range a {
		log.Printf("checking Label %s=%s if found on pod", k, v)
        if bv, ok := b[k]; !ok || bv != v {
            return false
        }
    }
    return true
}

func allowed() *admissionv1.AdmissionResponse {
    return &admissionv1.AdmissionResponse{Allowed: true}
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
