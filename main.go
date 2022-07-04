package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

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
	ProgramName         = "kove-service"
	deviceCheckInterval = 10 * time.Second
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
		klog.Info("YEV - failed to get kubeconfig", "error", err)
		os.Exit(2)
	}

	clientset, err := k8sclient.NewForConfig(kubeconfig)
	if err != nil {
		klog.Info("failed to get clientset", "error", err)
		os.Exit(2)
	}

	nodeName := os.Getenv("NODE_NAME")

	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, meta_v1.GetOptions{})
	if err != nil {
		klog.Info("failed to get node", "node", nodeName, "error", err)
		os.Exit(2)
	}

	for a := range node.Annotations {
		klog.Info("node annotations", "annotation", a)
	}

	patches := []JsonPatch{NewJsonPatch("add", "/status/capacity", "kove.net/memory", "1024Mi")}

	data, err := json.Marshal(patches)
	if err != nil {
		klog.Info("failed to marshal patches", "error", err)
		os.Exit(2)
	}

	_, err = clientset.CoreV1().Nodes().Patch(context.TODO(), nodeName, types.JSONPatchType, data, meta_v1.PatchOptions{}, "status")
	if err != nil {
		klog.Info("failed to patch node", "node", nodeName, "error", err)
		os.Exit(2)
	}

	i := 0
	for {
		klog.Info("In main loop", "index", i)
		i = i + 1
		output, err := generateKoveUtil()
		if err != nil {
			klog.Info("failed to generate kove memory from util ", "error ", err)
			os.Exit(2)
		}
		klog.Info(fmt.Sprintf("Kove util memory in bytes: %s", output))
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
	cmd := exec.Command("go", "run", "./util")
	return cmd.Output()
}
