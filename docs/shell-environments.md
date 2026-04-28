# Shell 环境文件对照

下面表格列出常见 shell 在不同阶段常用的配置文件：环境变量（Env/All）、交互配置（Interactive RC）、登录初始化（Login）与登出清理（Logout）。

| Shell | 环境变量 (Env/All) | 交互配置 (Interactive RC) | 登录初始化 (Login) | 登出清理 (Logout) |
|-------|---------------------|---------------------------|---------------------|-------------------|
| Zsh   | .zshenv             | .zshrc                   | .zprofile / .zlogin | .zlogout          |
| Bash (通常借用 .bashrc) | .bashrc | .bash_profile / .bash_login | .bash_logout | - |
| Fish  | config.fish (内判)  | config.fish (不常用，通过函数判断) | - | (需定义 on_exit 函数) |
| PowerShell | $PROFILE (所有宿主) | $PROFILE (当前宿主) | (同左) | (需在脚本块定义) |
| Nushell | env.nu | config.nu | (同左) | (不常用) |
| Dash  | $ENV 变量指向的文件 (不支持自动加载) | .profile (由 Login 进程加载) | - | (不支持) |

*注：不同系统/发行版与 shell 版本可能对这些文件的处理有差异，表中项仅供参考。*