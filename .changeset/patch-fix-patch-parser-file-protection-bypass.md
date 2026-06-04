---
"gh-aw": patch
---

Fixed a security issue where crafted patches could bypass file protection checks by exploiting parser differences between the patch parser and `git am`.
