.PHONY: clean CpuTempMqttClient lint yamllint

CpuTempMqttClient: main.go kubeClient/kubeClient.go
	go build -o $@ $<

clean:
	go clean
	$(RM) main 
	$(RM) CpuTempMqttClient

lint:
	golangci-lint run ./...
	go vet ./...

YAML=$(wildcard *.yaml)
yamllint: $(YAML)
	yamllint $(YAML)
	
