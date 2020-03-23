# mydocker
`mydocker`是一个简单的`Docker runC`实现

---

### 说明
不同`version`实现的功能是递增的
详情请看各个版本的`docs/$version.md`说明

---

### `v1` 版本介绍
- [x] 实现`cpu.cfs_period_us` 限制
- [x] 实现`cpu.cfs_quota_us` 限制
- [x] 实现`cpuset.cpus` 限制
- [x] 实现`memory.limit_in_bytes` 限制


### `v2` 版本介绍
- [x] 实现`namespace` 隔离
- [x] 实现`driver`层驱动(目前仅支持`overlay2`

### `v3` 版本介绍
- [ ] TODO