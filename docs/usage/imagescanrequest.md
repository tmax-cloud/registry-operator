# **ImageScanRequest resource**

## **What is it?**

ImageScanRequest represents the current image scanning state against the Clair which installed on with operator. It can also sending a scan report to elasticsearch and a user can view statistics from Kibana.

## How to create

### spec field

**Key**|**Requried**|**Type**|**Description**
:-----:|:-----:|:-----:|:-----:
scanTargets|Yes|[]scanTarget|The bunch of target to be scanned which under the same registry.

### spec.scanTargets field

**Key**|**Requried**|**Type**|**Description**
:-----:|:-----:|:-----:|:-----:
registryURL|Yes|string|The URL of container registry.
certifacateSecret|No|string|The secret name which contains a container registry's certificate keypair (TLS Secret recommended)
images|Yes|[]string|Image names to scan ('*' for all and '?' for regex can be used)
imagePullSecret|No|string|The secret name which contains a login credential for registry (Should be DockerConfigJson type)
insecure|No|bool|Allow insecure registry connection when using SSL
elasticSearch|No|bool|whether send vulunerability reports to elasticsearch

## Example

---

Scan each of images

```yaml
apiVersion: tmax.io/v1
kind: ImageScanRequest
metadata:
  name: 
spec:
scanTargets:
- registryUrl: "220.90.208.243"
  images: ["nginx:1.18.0","redis:5.0.10","golang:1.14.14","tomcat:8.5"]
  certificateSecret: poc-registry-cert
  imagePullSecret: hpcd-registry-tmax-registry
  elasticSearch: true
```

Scan all images in the repository

```yaml
apiVersion: tmax.io/v1
kind: ImageScanRequest
metadata:
  name:
spec:
scanTargets:
- registryUrl: "220.90.208.243"
  images: ["*"]
  certificateSecret: poc-registry-cert
  imagePullSecret: hpcd-registry-tmax-registry
  elasticSearch: true
```

## **Result**

---

The scan summary information is provided by resource's results or states.

You can also get visualized statistics from Kibana. It provide some basic template for your usage.

Or if you want to more detailed information, help your self by quering for your purpose. :)
