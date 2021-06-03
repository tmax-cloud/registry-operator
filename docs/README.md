# Contents

- [Installation](./install.md)
  - [Prerequisites](./install.md#prerequisites)
  - [Install procedure](./install.md#install-procedure)
  - [Test your installation](./install.md#test-your-installation)
  - [Tear down](./install.md#tear-down)
- [Environmental Variable Description](./envs.md)
- [Usage Of Resource](./usage/README.md)
- [About Developement](./dev.md)
- [About Release](./release.md)
- [How to push Image](./push.md)
- [How to configure Elasticsearch indexing for Image Scanning](https://tmaxcloud-ck1-2.s3.ap-northeast-2.amazonaws.com/%EC%9D%B4%EB%AF%B8%EC%A7%80%EC%8A%A4%EC%BA%94_Elasticsearch_%EC%9D%B8%EB%8D%B1%EC%8A%A4%EC%84%A4%EC%A0%95_%EA%B0%80%EC%9D%B4%EB%93%9C.pptx)
- [How to configure Elasticsearch alarming for Image Scanning](https://tmaxcloud-ck1-2.s3.ap-northeast-2.amazonaws.com/%EC%9D%B4%EB%AF%B8%EC%A7%80%EC%8A%A4%EC%BA%94_Elasticsearch_%EC%95%8C%EB%9E%8C%EC%84%A4%EC%A0%95_%EA%B0%80%EC%9D%B4%EB%93%9C.pptx)

## More Information

Released new notary image to use. The image has been distributed to the latest version of the bug modified. And modified server and signer's config. following URLs are forked github repository and released dockerhub repository.

- Github:
  - <https://github.com/tmax-cloud/notary/tree/v0.6.2-rc>

- Dockerhub:
  - <https://hub.docker.com/r/tmaxcloudck/notary_server>
    - latest version: v0.6.2-rc1
  - <https://hub.docker.com/r/tmaxcloudck/notary_signer>
    - latest version: v0.6.2-rc1
  - <https://hub.docker.com/r/tmaxcloudck/notary_mysql>
    - latest version: v0.6.2-rc2
