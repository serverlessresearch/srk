FROM lambci/lambda:provided

FROM nvidia/cuda

# Recreate lambci/lambda:base

RUN groupadd --gid 495 sandbox
RUN useradd -M --uid 496 --gid 495 --home-dir /home/sbx_user1051 --shell /sbin/nologin sbx_user1051

ENV PATH=/usr/local/bin:/usr/bin/:/bin:/opt/bin \
    LD_LIBRARY_PATH=/lib64:/usr/lib64:/var/runtime:/var/runtime/lib:/var/task:/var/task/lib:/opt/lib \
    LANG=en_US.UTF-8 \
    TZ=:UTC \
    LAMBDA_TASK_ROOT=/var/task \
    LAMBDA_RUNTIME_DIR=/var/runtime \
    _LAMBDA_CONTROL_SOCKET=14 \
    _LAMBDA_SHARED_MEM_FD=11 \
    _LAMBDA_LOG_FD=9 \
    _LAMBDA_SB_ID=7 \
    _LAMBDA_CONSOLE_SOCKET=16 \
    _LAMBDA_RUNTIME_LOAD_TIME=1530232235231 \
    _AWS_XRAY_DAEMON_ADDRESS=169.254.79.2 \
    _AWS_XRAY_DAEMON_PORT=2000 \
    AWS_XRAY_DAEMON_ADDRESS=169.254.79.2:2000 \
    AWS_XRAY_CONTEXT_MISSING=LOG_ERROR \
    _X_AMZN_TRACE_ID='Root=1-dc99d00f-c079a84d433534434534ef0d;Parent=91ed514f1e5c03b2;Sampled=1'

# Adapted from lambci/lambda:python3.7

ENV PATH=/var/lang/bin:$PATH \
    LD_LIBRARY_PATH=/var/lang/lib:$LD_LIBRARY_PATH \
    AWS_EXECUTION_ENV=AWS_Lambda_python3.7

RUN rm -rf /var/runtime /var/lang /var/rapid && \
  curl https://lambci.s3.amazonaws.com/fs/python3.8.tgz | tar -zx -C / var/runtime

RUN apt-get update && apt-get install -y python3.8 python3-pip \
    && rm -rf /var/lib/apt/lists/*

RUN pip3 install numpy

RUN mkdir -p /var/lang/bin \
    && ln -s /usr/bin/openssl /var/lang/bin/openssl \
    && ln -s /usr/bin/pip3 /var/lang/bin/pip3 \
    && ln -s /usr/bin/pip3 /var/lang/bin/pip \
    && ln -s /usr/bin/python3 /var/lang/bin/python3 \
    && ln -s /usr/bin/python3.8 /var/lang/bin/python3.8 \
    && ln -s /usr/bin/python3 /var/lang/bin/python \
    && ln -s /usr/bin/python3-config /var/lang/bin/python3-config \
    && ln -s /usr/bin/python3.8-config /var/lang/bin/python3.8-config


COPY --from=0 /var/runtime/init /var/rapid/init

USER sbx_user1051

ENTRYPOINT ["/var/rapid/init", "--bootstrap", "/var/runtime/bootstrap"]
