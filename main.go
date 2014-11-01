package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

var bind = flag.String("bind", ":8090", "the port on which the http server will bind")

type Neuron struct {
	Endpoint  string
	Container string
	Image     string
	Send      []*Neuron
}

func main() {

	//parse flags
	flag.Parse()

	//get config from env
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		log.Fatal(fmt.Errorf("Could not retrieve DOCKER_HOST, not provided as option and not in env"))
	}

	cpath := os.Getenv("DOCKER_CERT_PATH")
	if cpath == "" {
		log.Fatal(fmt.Errorf("Could not retrieve DOCKER_CERT_PATH, not provided as option and not in env"))
	}

	//change to http connection
	addr, err := url.Parse(host)
	if err != nil {
		log.Fatal(err)
	}

	addr.Scheme = "https"

	//setup tls docker client
	client, err := docker.NewTLSClient(addr.String(), filepath.Join(cpath, "cert.pem"), filepath.Join(cpath, "key.pem"), filepath.Join(cpath, "ca.pem"))
	if err != nil {
		log.Fatal(err)
	}

	//get hostname
	hn, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	//start a web server that logs incoming requests to a file
	log.Printf("%s, Listening for incoming on %s...", hn, *bind)
	err = http.ListenAndServe(*bind, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI != "/" {
			http.NotFound(w, r)
			return
		}

		//list running containers
		cs, err := client.ListContainers(docker.ListContainersOptions{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		//other neurons to choose from
		neurons := []*Neuron{}

		//randomly send new request to another neuron
		for _, c := range cs {

			//only consider other neurons
			if !strings.HasPrefix(c.Image, "neuron") {
				continue
			}

			//dont send to itself
			if strings.HasPrefix(c.ID, hn) {
				continue
			}

			//fetch public ports
			for _, p := range c.Ports {
				if p.PrivatePort != 8090 {
					continue
				}

				//create url for other neuron endpoint and add to collection
				ep := fmt.Sprintf("http://%s:%d", strings.SplitN(addr.Host, ":", 2)[0], p.PublicPort)
				neurons = append(neurons, &Neuron{ep, c.ID, c.Image, []*Neuron{}})
			}

		}

		//will we to next neuron proceed
		if len(neurons) > 0 && rand.Intn(3) < 2 {

			//if so which one
			n := neurons[rand.Intn(len(neurons))]

			//send request
			resp, err := http.Get(n.Endpoint)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			//decode the other neurons response (which should return its neurons etc)
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&n.Send)
			if err != nil {
				http.Error(w, "Decode other neuron:"+err.Error(), http.StatusInternalServerError)
				return
			}

		}

		//encode to json
		encoder := json.NewEncoder(w)
		err = encoder.Encode(neurons)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}))

	if err != nil {
		log.Fatal(err)
	}
}
