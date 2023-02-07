# ses-sidecar

## Usage

```
docker run -it \
 -p 1025:1025 \
 -e ADDR=0.0.0.0:1025 \
 -e AWS_ACCESS_KEY_ID \
 -e AWS_SECRET_ACCESS_KEY \
 -e AWS_SESSION_TOKEN \
 -e AWS_REGION \
 ghcr.io/aidansteele/ses-sidecar:latest
```

This will start an SMTP server listening on port 1025 that uses the AWS SES
SendRawEmail API to deliver email. In practice, you wouldn't pass credentials
like this example, you would associate an IAM role with the container via your 
orchestration system, e.g. an ECS task IAM role or EKS IRSA service account role. 
That role needs `ses:SendRawEmail` permission.

This is a proof-of-concept, but it works and can be deployed as a sidecar to
your application. It exists because (as of the time of writing) the SES SMTP 
service doesn't work with temporary credentials, which are a security best-practice.
File an issue if you have any problems / feature requests.
