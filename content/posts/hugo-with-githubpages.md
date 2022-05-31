---
title: "使用Hugo部署GithubPages"
date: 2022-05-29T16:08:13+08:00
draft: false
---

记录一下使用Hugo生成静态博客，并通过githubPages部署的过程。

这里，我的目标是：

1. 使用`blog-source`作为原始的内容仓库，`<your-name>.github.io`作为实际的githubPages仓库
2. 通过github Action将两者串联起来，原始内容提交变更时，自动触发内容生成并发布

这样的好处是，可以将`blog-source`作为私有仓库，并能直接以`<your-name>.github.io`作为URL。且通过github action实现CICD，解放双手实现自动化。这里我画了一张图，便于理解：

![img](https://raw.githubusercontent.com/erenming/image-pool/master/blog/flow.png)

# Hugo

安装Hugo，然后初始化

```bash
# macOS install hugo
brew install hugo

# create site project
hugo new site blog-source
```

选择你中意的主题并安装

```bash
cd blog-source
git init

# add paperMod as theme
git submodule add https://github.com/adityatelange/hugo-PaperMod themes/paperMod
```

添加文章并启动demo

```
hugo new posts/my-first-post.md

# start demo for preview
hugo server -D
```

创建一个额外的仓库，这里我创建一个名为blog-source的仓库并作为刚才创建的blog-source的远端仓库

```bash
cd blog-source
git init
git remote add origin <your-remove-git>
```

# GithubPages

创建一个githubPages仓库，名称必须是`<your-name>.github.io`。[DOC](https://docs.github.com/en/pages/getting-started-with-github-pages/creating-a-github-pages-site#creating-a-repository-for-your-site)

# Connection

创建sshKey:  `ssh-keygen -t rsa -b 4096 -C "$(git config user.email)" -f gh-pages -N ""`

将私钥`gh-pages`内容复制，并在`blog-source`仓库的`Settings->Secrets->Actions`创建secret变量

![img](https://github.com/peaceiris/actions-gh-pages/raw/main/images/secrets-1.jpg)

将公钥`gh-pages.pub`内容复制，并作为`<your-name>.github.io`的Deploy Key，**记得勾选读写权限**

![img](https://github.com/peaceiris/actions-gh-pages/raw/main/images/deploy-keys-1.jpg)

创建github workflow文件`.github/workflows/gh-pages.yml`，其内容如下所示：

```yaml
name: github pages

on:
  push:
    branches:
      - master  # Set a branch to deploy, my branch is master
  pull_request:

jobs:
  deploy:
    runs-on: ubuntu-20.04
    steps:
      - uses: actions/checkout@v2
        with:
          submodules: true  # Fetch Hugo themes (true OR recursive)
          fetch-depth: 0    # Fetch all history for .GitInfo and .Lastmod

      - name: Setup Hugo
        uses: peaceiris/actions-hugo@v2
        with:
          hugo-version: '0.99.1'
          # extended: true

      - name: Build
        run: hugo --minify

      - name: Deploy
        uses: peaceiris/actions-gh-pages@v3
        with:
          external_repository: <your-name>/<your-name>.github.io
          publish_branch: master
          # the secret key
          deploy_key: ${{ secrets.ACTIONS_DEPLOY_KEY }}
          publish_dir: ./public
```

完成后，在blog-source仓库提交代码并push即可触发workflow，可在仓库的Actions功能项下查看运行情况。

若无意外，稍等片刻(估计是因为Github同步并非完全实时)，即可通过`<your-name>/<your-name>.github.io`访问博客了

# 总结

综上所述，使用hugo生成静态网站，创建githubPages项目其实并不难，主要难点在于如何通过githubAction将两者链接起来，实现CICD。遇到问题，建议多多翻阅官方文档，一定是能解决的。

# 参考

- https://github.com/peaceiris/actions-gh-pages
- https://docs.github.com/en/pages/getting-started-with-github-pages/about-github-pages
- https://gohugo.io/hosting-and-deployment/hosting-on-github/
