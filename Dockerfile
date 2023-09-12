FROM alpine

WORKDIR /app
COPY tiebarankgo /app
RUN sed -i "s|dl-cdn.alpinelinux.org|mirrors.bfsu.edu.cn|" /etc/apk/repositories
RUN apk add --no-cache gcompat ca-certificates

CMD ["./tiebarankgo"]

EXPOSE 3000