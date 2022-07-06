package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/util/json"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

const (
	// ProgramName is the canonical name of this program
	ProgramName          = "kove-service"
	deviceCheckInterval  = 10 * time.Second
	extendedResourceName = "kove.net/memory"
	reduceFactor         = 150
)

type JsonPatch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value,omitempty"`
}

func main() {

	flags := flag.NewFlagSet(ProgramName, flag.ExitOnError)

	klog.InitFlags(flags)

	_ = flags.Parse(os.Args[1:])
	if len(flags.Args()) > 0 {
		fmt.Fprintf(flags.Output(), "unknown command line argument: %s\n", flags.Args()[0])
		flags.Usage()
		os.Exit(2)
	}

	kubeconfig, err := getKubeconfig()
	if err != nil {
		klog.Info("Failed to get kubeconfig", "error", err)
		os.Exit(2)
	}

	clientset, err := k8sclient.NewForConfig(kubeconfig)
	if err != nil {
		klog.Info("failed to get clientset", "error", err)
		os.Exit(2)
	}

	i := 0
	for {
		klog.Info("In main loop", "index", i)
		i = i + 1
		nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), meta_v1.ListOptions{LabelSelector: "!node-role.kubernetes.io/master"})
		if err != nil {
			klog.Info("Failed to list nodes in the cluster", "error", err)
			os.Exit(2)
		}

		for _, node := range nodeList.Items {
			quantity := node.Status.Allocatable.Name(extendedResourceName, resource.DecimalSI)
			var newVal int64
			var httpPatchAction string

			if quantity.Value() == math.MaxInt64 {
				// initial registration
				newVal = 1024
				httpPatchAction = "add"
			} else {
				httpPatchAction = "replace"
				if quantity.Value() < reduceFactor {
					newVal = 1024
				} else {
					newVal = quantity.Value() - reduceFactor
				}
			}
			klog.Info(fmt.Sprintf("Node: %s kove.net/memory | Before: %d  | After: %d", node.Name, quantity.Value(), newVal))

			strNewVal := strconv.FormatInt(newVal, 10)
			patches := []JsonPatch{NewJsonPatch(httpPatchAction, "/status/capacity", "kove.net/memory", strNewVal)}

			data, err := json.Marshal(patches)
			if err != nil {
				klog.Info("failed to marshal patches", "error", err)
				os.Exit(2)
			}

			_, err = clientset.CoreV1().Nodes().Patch(context.TODO(), node.Name, types.JSONPatchType, data, meta_v1.PatchOptions{}, "status")
			if err != nil {
				klog.Info("failed to patch node", "node", node.Name, "error", err)
				os.Exit(2)
			}
		}
		<-time.After(deviceCheckInterval)
	}
}

func getKubeconfig() (*restclient.Config, error) {
	return restclient.InClusterConfig()
}

func NewJsonPatch(verb string, jsonpath string, key string, value string) JsonPatch {
	return JsonPatch{verb, path.Join(jsonpath, strings.ReplaceAll(key, "/", "~1")), value}
}

func generateKoveUtil() ([]byte, error) {
	cmd := exec.Command("go", "run", "/util/kove.go")
	res, err := cmd.Output()
	fmt.Sprintf("Kove tmp memory: %s", string(res))
	return res, err
}
