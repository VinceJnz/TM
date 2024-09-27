# Notes

## Docker build

Docker files can only refer to local files that are in the same folder or sub-folders of the docker file.
To do a multi-stage build for the webserver and the wasm client the dockerfile needed to be able to access
both the client1 and the web-server folders

To do this the docker file needed to be located in the project folder so that the client1 and web-server folders were accssible sub-folders.
This was necessary in the multi-stage build to transfer files:
* wasm.main from wasmbuilder to finalbuilder
* webserver from webbuilder to finalbuilder
* htmel.index from webbuilder to finalbuilder

