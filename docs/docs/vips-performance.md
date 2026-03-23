# VIPS Performance Tuning

imagor uses [libvips](https://github.com/libvips/libvips) for image processing. libvips provides several configuration options to tune performance and resource usage:

`VIPS_CONCURRENCY` controls the number of threads libvips uses for image operations:

```dotenv
VIPS_CONCURRENCY=1    # Single-threaded (default)
VIPS_CONCURRENCY=-1   # Use all available CPU cores
VIPS_CONCURRENCY=4    # Use 4 threads
```

**Important:** `VIPS_CONCURRENCY` is a **global setting** that controls threading **within each image operation**, not the number of concurrent requests.

- **Default (1)**: Single-threaded processing. Recommended for most deployments where you handle concurrency at the application level (multiple imagor processes/containers).
- **-1 (auto)**: Uses all CPU cores. Can improve performance for individual large images but may cause resource contention under high request concurrency.
- **Custom value**: Set to a specific number of threads for fine-tuned control.

For high-traffic deployments, it's generally better to scale horizontally (more imagor instances) rather than increasing `VIPS_CONCURRENCY`.

libvips also has a built-in operation cache (`VIPS_MAX_CACHE_MEM`, `VIPS_MAX_CACHE_SIZE`, `VIPS_MAX_CACHE_FILES`) that reuses recently computed operations. For imagor's typical workload, each request processes a different source image so this cache rarely gets hits — the defaults (0 = disabled) are appropriate. See [libvips documentation](https://github.com/libvips/libvips/issues/1585) for details.
