package webssh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v4"
	clv1alpha2 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha2"
	"github.com/netgroup-polito/CrownLabs/operators/pkg/utils"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type clientInitMessage struct {
	Token     string `json:"token"`
	VMName    string `json:"vmName"`              // The name of the VM to connect to
	Namespace string `json:"namespace,omitempty"` // Optional namespace, can be derived from the token
}

// Global client and config
var (
	k8ClientCrd client.Client         // Global client for Kubernetes operations
	k8ClientAPI *kubernetes.Clientset // Kubernetes client for TokenReview API
	initOnce    sync.Once             // Ensures the client is initialized only once
	initErr     error                 // Holds any error that occurs during initialization
)

// NewK8sClient initializes the global k8s client.
func createClient() (client.Client, *kubernetes.Clientset, error) {
	// get the client for the CRD
	cCrd, err := utils.NewK8sClient()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	// Get the Kubernetes configuration
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("k8s config error: %w", err)
	}

	// Create a Kubernetes clientset for TokenReview API
	cApi, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	return cCrd, cApi, nil
}

// init initializes the global k8s client once
func initK8Client() {
	initOnce.Do(func() {
		var cCrd client.Client
		var cApi *kubernetes.Clientset
		cCrd, cApi, initErr = createClient()
		if initErr == nil {
			k8ClientCrd = cCrd
			k8ClientAPI = cApi
		}
	})
}

// GetInstance retrieves an Instance CR by namespace and name
func GetInstance(namespace, name string) (*clv1alpha2.Instance, error) {
	if initErr != nil {
		return nil, fmt.Errorf("client initialization failed: %w", initErr)
	}

	instance := &clv1alpha2.Instance{}
	err := k8ClientCrd.Get(context.TODO(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, instance)

	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return instance, nil
}

// ValidateToken checks if the provided token is valid via TokenReview API
func ValidateToken(token string) (*authv1.TokenReviewStatus, error) {
	if initErr != nil {
		return nil, fmt.Errorf("client initialization failed: %w", initErr)
	}

	review := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token: token,
		},
	}

	result, err := k8ClientAPI.AuthenticationV1().TokenReviews().Create(
		context.TODO(), review, metav1.CreateOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("token review failed: %w", err)
	}

	return &result.Status, nil
}

func extractUsernameFromToken(tokenString string) (string, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if username, ok := claims["preferred_username"].(string); ok {
			return username, nil
		}
	}
	return "", errors.New("username not found in token claims")
}

func validateRequest(firstMsg []byte, conf config) (string, error) {
	var initMsg clientInitMessage
	if err := json.Unmarshal(firstMsg, &initMsg); err != nil {
		return "", errors.New("invalid JSON format")
	}

	if initMsg.VMName == "" || initMsg.Token == "" {
		return "", errors.New("missing required fields in the initialization message")
	}

	// Extract username from the token
	username, err := extractUsernameFromToken(initMsg.Token)
	if err != nil {
		return "", errors.New("invalid token format: " + err.Error())
	}

	// Get the namespace from the message, or derive it from the token
	namespace := initMsg.Namespace
	if namespace == "" {
		namespace = "tenant-" + username
	}

	log.Printf("Validating request for user: %s, namespace: %s", username, namespace)

	// Validate the token
	status, err := ValidateToken(initMsg.Token)
	if err != nil {
		return "", errors.New("token validation failed: " + err.Error())
	}
	if !status.Authenticated {
		return "", errors.New("token is not authenticated")
	}

	log.Printf("Token is valid for user: %s", username)

	// get the instance by name and namespace
	instance, err := GetInstance(namespace, initMsg.VMName)
	if err != nil {
		return "", errors.New("failed to get instance: " + err.Error())
	}

	// Extract the IP address from the instance object
	if !instance.Spec.Running {
		return "", errors.New("instance is not running")
	}

	// extract the IP address from the instance status
	if instance.Status.IP == "" {
		return "", errors.New("instance has no IP address assigned")
	}

	return instance.Status.IP + ":" + conf.VMSSHPort, nil // Return the connection string (IP:port)
}
