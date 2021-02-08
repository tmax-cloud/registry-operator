# **RegistryJob resource**

## **What is it?**
RegistryJob is a job execution instance. Each RegistryJob is a history of a specific task.
The task can be external registry synchronization, image scan (TBD), image sign (TBD), etc...

## How to create

### spec field

**Key**|**Requried**|**Type**|**Description**
:-----:|:-----:|:-----:|:-----:
ttl|Yes|int|Time-to-live (in seconds) after the job's completion.<br/>0: Delete immediately.<br/>-1: Do not delete<br/>ttl>0: Delete after ttl seconds
priority|No|int|Priority of the job. It should be greater or equal to 0 (Default: 0)
syncRepository|No|registryJobSyncRepository|Repository Sync type task for an external registry

### spec.registryJobSyncRepository field

**Key**|**Requried**|**Type**|**Description**
:-----:|:-----:|:-----:|:-----:
externalRegistry.name|Yes|string|Name of the target external registry

## Example

---

Sync repositories for an external registry

```yaml
apiVersion: tmax.io/v1
kind: RegistryJob
metadata:
  name: job-sync-harbor-1
  namespace: default
spec:
  ttl: 0
  syncRepository:
    externalRegistry:
      name: harbor-1
```

## **Result**

---

For the repository sync task, you can check the list of the children Repository of the target external repository.
