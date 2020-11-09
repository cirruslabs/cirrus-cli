# podmanapi

The `podmanapi` package was generated using the Swagger Codegen 3.0.23 [downloaded from the Maven Repository](https://mvnrepository.com/artifact/io.swagger.codegen.v3/swagger-codegen-cli):

```
java -jar swagger-codegen-cli-3.0.23.jar generate -l go -i https://storage.googleapis.com/libpod-master-releases/swagger-latest-master.yaml -o podmanapi
```

The link to the [`swagger-latest-master.yaml`](https://storage.googleapis.com/libpod-master-releases/swagger-latest-master.yaml) was found in the auto-generated [API documentation page](https://podman.readthedocs.io/en/latest/_static/api.html).

Afterwards:

* the missing `os` imports were added
* `model_plugin_config_linux.go` was renamed to `model_plugin_config_linux_.go`

... to fix the compilation.
