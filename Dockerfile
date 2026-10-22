FROM golang:1.26.10-alpine3.24 AS build

WORKDIR /lambda/go
COPY  . ./
RUN CGO_ENABLED=0 go build -tags lambda.norpc -o main ./cmd/lambda/handler
RUN CGO_ENABLED=0 go build -tags lambda.norpc -o bedrock ./cmd/lambda/bedrock

# Build the final image from the Amazon Linux 2023 Lambda base image
FROM public.ecr.aws/lambda/provided:al2023.2026.10.22.03@sha256:612c98f126c2a95e77a016d56fed50a83055e11d636132ee6ebdd461f01b9617 AS aws

COPY --from=build /lambda/go/main /functions/main
COPY --from=build /lambda/go/bedrock /functions/bedrock
ENTRYPOINT [ "/functions/main" ]


# Install the Runtime Interface Emulator in the container image
FROM public.ecr.aws/lambda/provided:al2023.2026.10.22.03@sha256:612c98f126c2a95e77a016d56fed50a83055e11d636132ee6ebdd461f01b9617 AS local

COPY --from=build /lambda/go/main /functions/main
COPY --from=build /lambda/go/bedrock /functions/bedrock
ENTRYPOINT [ "/usr/local/bin/aws-lambda-rie" ]
CMD [ "/functions/main" ]
