running neuron on boot2docker
`docker run -P -it -e DOCKER_HOST=tcp://`boot2docker ip`:2376 -e DOCKER_CERT_PATH=/cert -v /home/docker/.docker:/cert neuron`