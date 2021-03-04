# **ImageScanRequest resource**

## **What is it?**

ImageScan Request displays a list of images that need to be scanned from the image scan server and the processed results. It can also send the scan reports to elasticsearch server.

## How to create

### spec field

**Key**|**Requried**|**Type**|**Description**
:-----:|:-----:|:-----:|:-----:
scanTargets|Yes|[]scanTarget|The bunch of target to be scanned which under the same registry.
insecure|No|bool|Do not verify registry server's certificate
sendReport|No|bool|Whether to send result to report server(elasticsearch)
maxFixable|No|bool|The number of fixable issues allowable

### spec.scanTargets field

**Key**|**Requried**|**Type**|**Description**
:-----:|:-----:|:-----:|:-----:
registryURL|Yes|string|The image registry address.
certifacateSecret|No|string|The name of certificate secret for private registry. If secret is 'Opaque' type, the key of certificate should be 'ca.crt' or 'tls.crt';TLS type secret is recommended
images|Yes|[]string|Image names to scan ('*' for all and '?' for regex can be used)
imagePullSecret|No|string|The name of secret containing login credential of registry (The secret should be 'DockerConfigJson' type)

## Example

---

Scan each of images

```yaml
apiVersion: tmax.io/v1
kind: ImageScanRequest
metadata:
  name: poc-base-images
spec:
scanTargets:
- registryUrl: "220.90.208.243"
  images: ["nginx:1.18.0","redis:5.0.10","golang:1.14.14","tomcat:8.5"]
  certificateSecret: poc-registry-cert
  imagePullSecret: hpcd-registry-tmax-registry
sendReport: true
```

Scan all images in the repository

```yaml
apiVersion: tmax.io/v1
kind: ImageScanRequest
metadata:
  name: poc-all
spec:
scanTargets:
- registryUrl: "220.90.208.243"
  images: ["*"]
  certificateSecret: poc-registry-cert
  imagePullSecret: hpcd-registry-tmax-registry
sendReport: true
```

## **Result**

---

The scan summary information is provided by resource's results or states.

You can also get visualized statistics from Kibana. It provide some basic template for your usage.

Or if you want to more detailed information, help your self by quering for your purpose. :)
