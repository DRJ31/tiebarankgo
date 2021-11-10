from ubuntu

WORKDIR /app
COPY tiebarankgo /app
RUN apt update
RUN apt install ca-certificates

CMD ["./tiebarankgo"]