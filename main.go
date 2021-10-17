package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"

	cc "github.com/logrusorgru/aurora"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

//define global variables to be sent to the functions getnodes and exec to nodes.
var userSelectedIndex int
var userSelectedNode string
var nodeName []string

type sizeQueue chan remotecommand.TerminalSize

//Display usage message if user had syntax error.
const usage = `
Usage
--------------
For this tool to work, you should have a valid kubeconfig file located on /home/<user>/.kube/config.

Interactive:
./kubego

Non-Interactive:
./kubego [nodeName]

Note: Kubego does not require kubectl to be installed.`

func main() {
	//First check if the user provided a nodename. if yes, send the node to exec funcation.
	if len(os.Args) == 2 && os.Args[1] != "-h" {
		nodeName := os.Args[1]
		userSelectedNode = nodeName
		fmt.Printf("You've slected node: %s\n", cc.Yellow(nodeName))
		fmt.Printf("Verifying if node %s exist in your cluster\n", cc.Yellow(nodeName))
		verifyNode(userSelectedNode)
		execToNode(userSelectedNode)
		//if the user did not provide arguments to the command. use client-go to get the nodes and prompt the user to select.
		//then return the node to exec function.
	} else if len(os.Args) == 1 {
		fmt.Printf("No Node selected, please select the node number..\n")
		getNodes() //here we are calling exec get node and storing the value returned by the function to variable userSelectedNode
		fmt.Println()
		fmt.Printf("You've selected node: %s\n", cc.Yellow(userSelectedNode))
		execToNode(userSelectedNode) //here we are calling exec function and passing the value coming from getnodes function.
	} else if os.Args[1] == "-h" {
		fmt.Println(usage)
	} else {
		fmt.Println(usage)
	}
}

func verifyNode(n string) {
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

	nodes, err := coreclient.Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	//use range function to find the nodes, var countNode has the node index and var node has string of node name.
	for _, node := range nodes.Items {
		nodeName = append(nodeName, node.Name)
	}

	//compare if the node exist in the slice, if not, exit the script. if yes, return the value and go to exec function.
	if !nodeExist(nodeName, userSelectedNode) {
		fmt.Printf("Node %s was not found in your cluster\n", cc.Red(userSelectedNode))
		os.Exit(1)
	}
	fmt.Printf("Node %s found, starting a shell.. \n", cc.Blue(userSelectedNode))

}

func getNodes() string {
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

	nodes, err := coreclient.Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	//use range function to find the nodes, var countNode has the node index and var node has string of node name.
	for countNode, node := range nodes.Items {
		fmt.Printf("%d %s\n", cc.BgBlue(countNode), cc.Blue(node.Name))
		nodeName = append(nodeName, node.Name)
	}

	//use scanln to get the user input and return the node name for that index in the slice.
	fmt.Scanln(&userSelectedIndex)
	userSelectedNode = nodeName[userSelectedIndex]
	return userSelectedNode
}

func execToNode(n string) {

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
	podName := "kubego-" + userSelectedNode + "-" + time.Now().Format("20060102150405")
	pod, err := coreclient.Pods(namespace).Create(context.TODO(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
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
			NodeName:                      userSelectedNode,
			HostPID:                       true,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}

	// Delete the Pod before we exit.
	defer coreclient.Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})

	// Wait for the Pod to indicate Ready == True.
	watcher, err := coreclient.Pods(namespace).Watch(context.TODO(),
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

	////////////////
	// Put the terminal into raw mode to prevent it echoing characters twice.
	oldState, err := term.MakeRaw(0)
	if err != nil {
		panic(err)
	}

	termWidth, termHeight, _ := term.GetSize(0)
	termSize := remotecommand.TerminalSize{Width: uint16(termWidth), Height: uint16(termHeight)}
	s := make(sizeQueue, 1)
	s <- termSize

	defer func() {
		err := term.Restore(0, oldState)
		if err != nil {
			panic(err)
		}
	}()

	///////////////

	// Connect this process' std{in,out,err} to the remote shell process.
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:             os.Stdin,
		Stdout:            os.Stdout,
		Stderr:            os.Stderr,
		Tty:               true,
		TerminalSizeQueue: s,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Deleting %q pod\n\r", cc.Red(podName))
}

func nodeExist(slice []string, find string) bool {
	for _, s := range slice {
		if s == find {
			return true
		}
	}
	return false
}

func (s sizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-s
	if !ok {
		return nil
	}
	return &size
}
