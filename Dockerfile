FROM debian:jessie

COPY kafkafeeder_*.deb heka_*.deb /tmp/

RUN dpkg -i /tmp/heka_*.deb \
    && dpkg -i /tmp/kafkafeeder_*.deb

VOLUME [ \
    "/www/kafkafeeder/conf/", \
    "/www/kafkafeeder/heka/", \
    "/www/kafkafeeder/logs/", \
    "/www/kafkafeeder/run/", \
    "/www/kafkafeeder/self-logs/" \
]

EXPOSE 8796

ENTRYPOINT [ "/www/kafkafeeder/bin/kafkafeeder" ]
CMD [ "-c", "/www/kafkafeeder/conf/conf.yaml" ]

