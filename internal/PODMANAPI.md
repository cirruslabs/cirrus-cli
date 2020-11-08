# podmanapi

The `podmanapi` package was generated using the [swagger-codegen-cli-2.4.17.jar](https://oss.sonatype.org/content/repositories/releases/io/swagger/swagger-codegen-cli/2.4.17/swagger-codegen-cli-2.4.17.jar) and the following command:

```
java -jar swagger-codegen-cli-2.4.17.jar generate -l go -i https://storage.googleapis.com/libpod-master-releases/swagger-latest-master.yaml -o podmanapi
```

The link to the [`swagger-latest-master.yaml`](https://storage.googleapis.com/libpod-master-releases/swagger-latest-master.yaml) was found in the auto-generated [API documentation page](https://podman.readthedocs.io/en/latest/_static/api.html).

The following files were edited to fix the `invalid operation: cannot take address of localVarOptionals.Request.Value() (value of type string)` compilation error:

```diff
diff --git a/internal/podmanapi/api_containers.go b/internal/podmanapi/api_containers.go
index 4d82a86..1c64efe 100644
--- a/internal/podmanapi/api_containers.go
+++ b/internal/podmanapi/api_containers.go
@@ -1953,7 +1953,7 @@ func (a *ContainersApiService) LibpodPlayKube(ctx context.Context, localVarOptio
 	}
 	// body params
 	if localVarOptionals != nil && localVarOptionals.Request.IsSet() {
-		localVarPostBody = &localVarOptionals.Request.Value()
+		localVarPostBody = localVarOptionals.Request.Value()
 		
 	}
 	r, err := a.client.prepareRequest(ctx, localVarPath, localVarHttpMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, localVarFileName, localVarFileBytes)
@@ -2176,7 +2176,7 @@ func (a *ContainersApiService) LibpodPutArchive(ctx context.Context, name string
 	}
 	// body params
 	if localVarOptionals != nil && localVarOptionals.Request.IsSet() {
-		localVarPostBody = &localVarOptionals.Request.Value()
+		localVarPostBody = localVarOptionals.Request.Value()
 		
 	}
 	r, err := a.client.prepareRequest(ctx, localVarPath, localVarHttpMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, localVarFileName, localVarFileBytes)
diff --git a/internal/podmanapi/api_containers_compat.go b/internal/podmanapi/api_containers_compat.go
index 6fcef27..26c8fdb 100644
--- a/internal/podmanapi/api_containers_compat.go
+++ b/internal/podmanapi/api_containers_compat.go
@@ -1626,7 +1626,7 @@ func (a *ContainersCompatApiService) PutArchive(ctx context.Context, name string
 	}
 	// body params
 	if localVarOptionals != nil && localVarOptionals.Request.IsSet() {
-		localVarPostBody = &localVarOptionals.Request.Value()
+		localVarPostBody = localVarOptionals.Request.Value()
 		
 	}
 	r, err := a.client.prepareRequest(ctx, localVarPath, localVarHttpMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, localVarFileName, localVarFileBytes)
diff --git a/internal/podmanapi/api_images_compat.go b/internal/podmanapi/api_images_compat.go
index 4995682..5dab6ad 100644
--- a/internal/podmanapi/api_images_compat.go
+++ b/internal/podmanapi/api_images_compat.go
@@ -329,7 +329,7 @@ func (a *ImagesCompatApiService) CreateImage(ctx context.Context, localVarOption
 	}
 	// body params
 	if localVarOptionals != nil && localVarOptionals.Request.IsSet() {
-		localVarPostBody = &localVarOptionals.Request.Value()
+		localVarPostBody = localVarOptionals.Request.Value()
 		
 	}
 	r, err := a.client.prepareRequest(ctx, localVarPath, localVarHttpMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, localVarFileName, localVarFileBytes)
@@ -660,7 +660,7 @@ func (a *ImagesCompatApiService) ImportImage(ctx context.Context, localVarOption
 	}
 	// body params
 	if localVarOptionals != nil && localVarOptionals.Request.IsSet() {
-		localVarPostBody = &localVarOptionals.Request.Value()
+		localVarPostBody = localVarOptionals.Request.Value()
 		
 	}
 	r, err := a.client.prepareRequest(ctx, localVarPath, localVarHttpMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, localVarFileName, localVarFileBytes)
diff --git a/internal/podmanapi/api_pods.go b/internal/podmanapi/api_pods.go
index ab284e2..9c2ce5e 100644
--- a/internal/podmanapi/api_pods.go
+++ b/internal/podmanapi/api_pods.go
@@ -687,7 +687,7 @@ func (a *PodsApiService) LibpodPlayKube(ctx context.Context, localVarOptionals *
 	}
 	// body params
 	if localVarOptionals != nil && localVarOptionals.Request.IsSet() {
-		localVarPostBody = &localVarOptionals.Request.Value()
+		localVarPostBody = localVarOptionals.Request.Value()
 		
 	}
 	r, err := a.client.prepareRequest(ctx, localVarPath, localVarHttpMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, localVarFileName, localVarFileBytes)
```
