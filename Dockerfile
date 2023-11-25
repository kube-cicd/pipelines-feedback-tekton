FROM alpine:3.18 AS builder
COPY .build/pipelines-feedback-tekton /pipelines-feedback-tekton
RUN chmod +x /pipelines-feedback-tekton

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /pipelines-feedback-tekton /pipelines-feedback-tekton

WORKDIR "/"
USER 65161
ENTRYPOINT ["/pipelines-feedback-tekton"]
