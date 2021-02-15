FROM golang:1.15-alpine

RUN apk update \
  && apk upgrade \
  && apk add --no-cache git \
    shadow \
    openssh \
    openssh-sftp-server \
    gcc \
    g++ \
    make \
    nfs-utils \
    zsh \
  && ssh-keygen -t rsa -f /etc/ssh/ssh_host_rsa_key \
  && echo "root:root" | chpasswd \
  && sh -c "$(wget https://raw.github.com/ohmyzsh/ohmyzsh/master/tools/install.sh -O -)" \
  && git clone git://github.com/zsh-users/zsh-autosuggestions \
    /root/.oh-my-zsh/custom/plugins/zsh-autosuggestions \
  && git clone git://github.com/zsh-users/zsh-syntax-highlighting.git \
    /root/.oh-my-zsh/custom/plugins/zsh-syntax-highlighting

RUN go get github.com/go-delve/delve/cmd/dlv

COPY sshd_config /etc/ssh/sshd_config
COPY zshrc /root/.zshrc

EXPOSE 22
EXPOSE 2345

CMD /usr/sbin/sshd -De

