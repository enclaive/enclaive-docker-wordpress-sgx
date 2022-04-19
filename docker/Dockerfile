from ubuntu:20.04



RUN apt update &&\
    DEBIAN_FRONTEND=noninteractive \
    apt install -y build-essential pkg-config autoconf libtool clang-10 clang-12 cmake doxygen gdb libssl-dev ninja-build \
    musl-dev musl python3 wget software-properties-common less vim &&\
    wget -qO- https://download.01.org/intel-sgx/sgx_repo/ubuntu/intel-sgx-deb.key | apt-key add &&\
    add-apt-repository "deb [arch=amd64] https://download.01.org/intel-sgx/sgx_repo/ubuntu `lsb_release -cs` main" &&\
    wget https://github.com/edgelesssys/edgelessrt/releases/download/v0.3.0/edgelessrt_0.3.0_amd64.deb &&\
    apt install -y ./edgelessrt_0.3.0_amd64.deb build-essential libssl-dev &&\
    wget https://github.com/edgelesssys/ego/releases/download/v0.4.2/ego_0.4.2_amd64.deb &&\
    apt install -y ./ego_0.4.2_amd64.deb build-essential libssl-dev &&\
    rm -rf /var/lib/apt/lists/* &&\
    wget https://go.dev/dl/go1.17.9.linux-amd64.tar.gz &&\
    tar -C /usr/local/ -xzf go1.17.9.linux-amd64.tar.gz
