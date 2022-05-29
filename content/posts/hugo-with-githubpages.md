---
title: "Hugo With Githubpages"
date: 2022-05-29T16:08:13+08:00
draft: true
---

记录一下使用Hugo生成静态博客，并通过githubPages部署的过程。

这里，我们的目标是：

1. 使用`blog-source`作为原始的内容仓库，`<your-name>.github.io`作为实际的githubPages仓库
2. 通过github Action将两者串联起来，原始内容提交变更时，自动触发内容生成并发布

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
          deploy_key: ${{ secrets.ACTIONS_DEPLOY_KEY }}
          publish_dir: ./public
```



# 参考

- https://github.com/peaceiris/actions-gh-pages
- https://docs.github.com/en/pages/getting-started-with-github-pages/about-github-pages
- https://gohugo.io/hosting-and-deployment/hosting-on-github/
