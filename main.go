package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh/terminal"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

func main() {
	//First check if the user provided a node, otherwise, get the nodes and let the user choose.
	if len(os.Args) == 2 {
		nodeName := os.Args[1]
		fmt.Printf("You've slected node: %s\n", nodeName)
		fmt.Printf("Checking if node: %s exists and accessible using your default kubeconfig file\n", nodeName)
	} else if len(os.Args) == 1 {
		fmt.Printf("No Node selected, please select the node number..\n")
		getNodes()
	} else {
		fmt.Println(usage)
	}
}

//Display usage message if user had syntax error.
const usage = `
Usage
--------------
For this tool to work, you should have a valid kubeconfig file.

Interactive:
./kubego [nodeName]

Non-Interactive:
./kubego

Note: Kubego does not require kubectl to be installed.`

func getNodes() {
	// Instantiate loader for kubeconfig file.
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	// Get a rest.Config from the kubeconfig file.  This will be passed into all
	// the client objects we create.
	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		panic(err)
	}

	// Create a Kubernetes core/v1 client.
	coreclient, err := corev1client.NewForConfig(restconfig)
	if err != nil {
		panic(err)
	}

	nodes, err := coreclient.Nodes().List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	//use range function to find the nodes, countNode has the node index and node has string of node name.
	for countNode, node := range nodes.Items {
		fmt.Printf("%d %s\n", countNode, node.Name)
	}
}

func execToNode() {

	// Instantiate loader for kubeconfig file.
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	// Determine the Namespace referenced by the current context in the
	// kubeconfig file.
	namespace, _, err := kubeconfig.Namespace()
	if err != nil {
		panic(err)
	}

	// Get a rest.Config from the kubeconfig file.  This will be passed into all
	// the client objects we create.
	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		panic(err)
	}

	// Create a Kubernetes core/v1 client.
	coreclient, err := corev1client.NewForConfig(restconfig)
	if err != nil {
		panic(err)
	}

	// Create a alpine Pod.  By running `cat`, the Pod will sit and do nothing.
	var privi bool = true
	var zero int64
	pod, err := coreclient.Pods(namespace).Create(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "busybox",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "busybox",
					Image: "busybox",
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privi,
					},
					Command:   []string{"cat"},
					Stdin:     true,
					StdinOnce: true,
					TTY:       true,
				},
			},
			TerminationGracePeriodSeconds: &zero,
			HostPID:                       true,
		},
	})
	if err != nil {
		panic(err)
	}

	// Delete the Pod before we exit.
	defer coreclient.Pods(namespace).Delete(pod.Name, &metav1.DeleteOptions{})

	// Wait for the Pod to indicate Ready == True.
	watcher, err := coreclient.Pods(namespace).Watch(
		metav1.SingleObject(pod.ObjectMeta),
	)
	if err != nil {
		panic(err)
	}

	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Modified:
			pod = event.Object.(*corev1.Pod)

			// If the Pod contains a status condition Ready == True, stop
			// watching.
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady &&
					cond.Status == corev1.ConditionTrue {
					watcher.Stop()
				}
			}

		default:
			panic("unexpected event type " + event.Type)
		}
	}

	// Prepare the API URL used to execute another process within the Pod.  In
	// this case, we'll run a remote shell.
	req := coreclient.RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pod.Spec.Containers[0].Name,
			Command:   []string{"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid", "--", "bash", "-l"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restconfig, "POST", req.URL())
	if err != nil {
		panic(err)
	}

	// Put the terminal into raw mode to prevent it echoing characters twice.
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(0, oldState)

	// Connect this process' std{in,out,err} to the remote shell process.
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    true,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Deleting kubego pod\n\r")
}
