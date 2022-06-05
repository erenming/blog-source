---
title: "Deploy blog with Hugo and GithubPages"
date: 2022-05-29T16:08:13+08:00
draft: false
---

Record the procedure of generate static blog with *Hugo*, and deploy on githubPages automatically.

Here is my goals:
1. `blog-source` as the source content repo, `<your-name>.github.io` as the actually githubPages repo
2. Connect above two repos with Github Action, trigger content generation and deploy when code pushed in `blog-source`

The advantages is as follows:
- Make the `blog-soursum ce` as private repo, and with `<your-name>.github.io` as your blog URL.
- Implement CICD with GitHub Action

I create a diagram to show this idea.
![img](https://raw.githubusercontent.com/erenming/image-pool/master/blog/flow.png)

# Hugo

安装Hugo，然后初始in
Install Hugo, and initialize. 

```bash
# macOS install hugo
brew install hugo

# create site project
hugo new site blog-source
```

Select your favorite theme and activate.

```bash
cd blog-source
git init

# add paperMod as theme
git submodule add https://github.com/adityatelange/hugo-PaperMod themes/paperMod
```

Add a article and start demo.

```
hugo new posts/my-first-post.md

# start demo for preview
hugo server -D
```

Create a repo named `blog-source` as the remote repo of site blog-source newly created.

```bash
cd blog-source
git init
git remote add origin <your-remove-git>
```

# GithubPages

Create a githubPages repo, the name must be `<your-name>.github.io`. You could refer this [document](https://docs.github.com/en/pages/getting-started-with-github-pages/creating-a-github-pages-site#creating-a-repository-for-your-site) for detail 

# Connection

Create sshKey: `ssh-keygen -t rsa -b 4096 -C "$(git config user.email)" -f gh-pages -N ""`

Copy private key `gh-pages`, and as the value of secret variable which created in `blog-source` repo's `Settings->Secrets->Actions` 


![img](https://github.com/peaceiris/actions-gh-pages/raw/main/images/secrets-1.jpg)

Copy public key `gh-pages.pub` as Deploy Key of ``, and remember to check Read/Write permission.

![img](https://github.com/peaceiris/actions-gh-pages/raw/main/images/deploy-keys-1.jpg)

Create GitHub workflow file `.github/workflows/gh-pages.yml`, the content as follows:

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

After all completion, push code in blog-source which will trigger workflow, you could check the running status at the repo's Action Page.

If there isn't any exception, you can visit your blog with the URL `<your-name>/<your-name>.github.io` after a while.

# 总结
In conclusion, it's easy to generate a static website or create a githubPages repo. The crux is the connection of the two repo and CICD.
Always refer the official document when encounter problem.

# 参考

- https://github.com/peaceiris/actions-gh-pages
- https://docs.github.com/en/pages/getting-started-with-github-pages/about-github-pages
- https://gohugo.io/hosting-and-deployment/hosting-on-github/
