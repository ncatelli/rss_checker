# Container image that runs your code
FROM ghcr.io/ncatelli/rss_checker:0.1.0

COPY entrypoint.sh /entrypoint.sh

# Code file to execute when the docker container starts up (`entrypoint.sh`)
ENTRYPOINT ["/entrypoint.sh"]
