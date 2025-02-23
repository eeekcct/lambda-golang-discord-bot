FROM golang:1.23.2-alpine3.20 AS build

WORKDIR /lambda/go
COPY  . ./
RUN CGO_ENABLED=0 go build -tags lambda.norpc -o main ./cmd/lambda/handler
RUN CGO_ENABLED=0 go build -tags lambda.norpc -o bedrock ./cmd/lambda/bedrock

# Build the final image from the scratch image
FROM public.ecr.aws/lambda/provided:al2.2024.10.16.13 AS aws

COPY --from=build /lambda/go/main /functions/main
COPY --from=build /lambda/go/bedrock /functions/bedrock
ENTRYPOINT [ "/functions/main" ]


# Install the Runtime Interface Emulator in the container image
FROM public.ecr.aws/lambda/provided:al2.2024.10.16.13 AS local

COPY --from=build /lambda/go/main /functions/main
COPY --from=build /lambda/go/bedrock /functions/bedrock
ENTRYPOINT [ "/usr/local/bin/aws-lambda-rie" ]
CMD [ "/functions/main" ]
