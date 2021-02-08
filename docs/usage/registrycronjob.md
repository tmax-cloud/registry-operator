# **RegistryCronJob resource**

## **What is it?**

RegistryCronJob periodically creates RegistryJob, like CronJob.

## How to create

### spec field

**Key**|**Requried**|**Type**|**Description**
:-----:|:-----:|:-----:|:-----:
schedule|Yes|string|The bunch of target to be scanned which under the same registry. Refer to https://ko.wikipedia.org/wiki/Cron for the spec.
jobSpec|Yes|registryJobSpec|Job spec for the created RegistryJobs

### spec.jobSpec field
Refer to [RegistryJob](./registryjob.md#spec-field)

## Example

---

Periodically sync the repositories of an external registry

```yaml
apiVersion: tmax.io/v1
kind: RegistryCronJob
metadata:
  name: rcj-sync-harbor-1
spec:
  schedule: "*/1 * * * *"
  jobSpec:
    ttl: -1
    syncRepository:
      externalRegistry:
        name: harbor-1
```

## **Result**

---

You can check the list of the RegistryJobs which are periodically created
