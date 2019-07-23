package main

import (
        //"fmt"
        "time"
        "sync"
        "bytes"
        //"strings"
        // "k8s.io/apimachinery/pkg/api/errors"
        v1 "k8s.io/api/core/v1"
        metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
        watch "k8s.io/apimachinery/pkg/watch"
        "k8s.io/client-go/kubernetes"
        "k8s.io/client-go/rest"
        log "github.com/sirupsen/logrus"
        "net/http"
)

func main() {

    config, err := rest.InClusterConfig()
    if err != nil {
        panic(err.Error())
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        panic(err.Error())
    }

    pods, err := clientset.CoreV1().Pods("default").List(metav1.ListOptions{ LabelSelector: "watch=need" }) // get all pods need to be watched
    if err != nil {
        panic(err.Error())
    }

    log.Infof("%d pods need to be watched\n", len(pods.Items))

    pods_to_watch := []*v1.Pod{}
    for _, item := range pods.Items {
        pods_to_watch = append(pods_to_watch, &item)
    }

    event_listener, err := clientset.CoreV1().Pods("default").Watch(metav1.ListOptions{ LabelSelector: "watch=need" })
    if err != nil {
        panic(err.Error())
    }

    var mtx sync.Mutex

    go func() {
      for e := range event_listener.ResultChan() {
          if e.Type == watch.Deleted { // remove pod from watch list
              pod := e.Object.(*v1.Pod)
              for i, item := range pods_to_watch {
                  if item.GetName() == pod.GetName() {
                       mtx.Lock()
                       pods_to_watch = append(pods_to_watch[:i], pods_to_watch[i+1:]...)
                       log.Infof("remove pod %s from watch queue", e.Object.(*v1.Pod).GetName())
                       mtx.Unlock()
                       break
                  }
              }
          }
      }
    }()

    log.Info("watching starts...")
    for {
        mtx.Lock()
        for _, pod := range pods_to_watch {
           if pod.Status.Phase == "Running" {
               var buffer bytes.Buffer
               buffer.WriteString("http://")
               buffer.WriteString(pod.Status.PodIP)
               buffer.WriteString(":")
               buffer.WriteString(pod.Labels["port"])
               buffer.WriteString("/?ping=")
               buffer.WriteString(pod.Labels["ping"]) // read endpoint for ping purpose
               ping_url := buffer.String()
               log.Infof("ping url: %s", ping_url)
               _, err := http.Get(ping_url) // ping, then reset the pod on failure, will set a configurabe threshold
               if err != nil {
                   // pod.Reset()
                   panic(err.Error())
                   log.Infof("reset pod %s due to ping failture", pod.GetName())
               } else {
                   log.Infof("ping passed, pod %s is fine.", pod.GetName())
               }
           } else if pod.Status.Phase == "Failed" {
               //pod.Reset()
               log.Infof("reset pod %s due to stauts failture", pod.GetName())
           } else {
               log.Infof("pod: %s, phase: %s", pod.GetName(), pod.Status.Phase)
           }
        }
        mtx.Unlock()
        time.Sleep(30 * time.Second) // will make interval configurable
    }
}
