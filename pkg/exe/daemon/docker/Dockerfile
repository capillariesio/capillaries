FROM golang:1.26.1 AS build
WORKDIR /usr/src/capillaries
ADD go.mod go.sum ./
ADD ./pkg ./pkg
RUN go mod download && go mod verify
RUN CGO_ENABLED=0 go build -o /usr/local/bin/capidaemon -ldflags="-s -w -X main.version=docker-$(date -u +'%Y-%m-%dT%H:%M:%SZ')" /usr/src/capillaries/pkg/exe/daemon

FROM python:3.12-alpine
# FROM gcr.io/distroless/python3-debian12 no pip
# FROM python:3.12-slim 2 times bigger than alpine, still DNS errors
# Portfolio test needs dateutil
RUN pip install python-dateutil
COPY --from=build /usr/local/bin/capidaemon /usr/local/bin/ 
ADD ./pkg/exe/daemon/capidaemon.json /usr/local/bin/

# nsjail
# General considerations. Consider this python script call on your devbox. This suceeds:

# /usr/bin/python3 -u - <<EOF
# from urllib.request import urlopen
# try:
#   with urlopen("https://www.google.com") as response:
#     print(response.read()[0:200].decode("utf-8"))
# except Exception as e:
#     print(f"Exception: {e}")
# EOF

# and this fails:

# /usr/local/bin/nsjail -Mo --chroot / -- /usr/bin/python3 -u - <<EOF
# from urllib.request import urlopen
# try:
#   with urlopen("https://www.google.com") as response:
#     print(response.read()[0:200].decode("utf-8"))
# except Exception as e:
#     print(f"Exception: {e}")
# EOF

# If you are using nsjail for your tests, copy it, so you can use it in your capidaemon.json Python settings.
# Make sure it is statically linked (no lib dependencies), in Makefile:
# LDFLAGS += -static -pie -Wl,-z,noexecstack -lpthread $(PROTOBUF_LIBS)
# so "file nsjail" says "statically linked", "ldd nsjail" says "not a dynamic executable".
# In capidaemon.json, configure Python interpreter like this:
#   "custom_processors": {
#     "py_calc": {
#       "python_interpreter_path": "/usr/local/bin/nsjail",
#       "python_interpreter_params": [
#         "-Mo", "--chroot", "/", "--user", "99999", "--group", "99999", "-R", "/usr", "-R", "/lib", "-R", "/lib64", "-R", "/bin", "-R", "/sbin", "--", "/usr/local/bin/python3", "-u", "-"
#       ]
#     }, ...
# If running in docker test environmen, set seccomp=unconfined so nsjail can call clone()
# security_opt:
#  - seccomp=unconfined

# COPY ./build/linux/amd64/nsjail /usr/local/bin/
# RUN chmod +x /usr/local/bin/nsjail

ENTRYPOINT ["/usr/local/bin/capidaemon"]


