FROM ubuntu

WORKDIR /app
COPY tiebarankgo /app
RUN apt update
RUN apt install -y ca-certificates

CMD ["./tiebarankgo"]

EXPOSE 3000