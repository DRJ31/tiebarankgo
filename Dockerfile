FROM alpine

WORKDIR /app
COPY tiebarankgo /app
RUN apk add --no-cache gcompat ca-certificates

CMD ["./tiebarankgo"]

EXPOSE 3000