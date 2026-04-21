#!/bin/bash

# Go 代码行数统计 - 支持 --latest / --all / --day
# 用法：
# ./xglcc.sh --latest    查看最近一次提交后的累计有效总行数
# ./xglcc.sh --all       查看所有提交历史的累计总行数（前缀和）
# ./xglcc.sh --day       按天查看代码量趋势（每天的累计总行数）

if [ "$1" = "--latest" ]; then
    # 最近一次提交：最终累计总行数
    git log --all --pretty=tformat: --numstat -- '*.go' | awk '{ add += $1; subs += $2 } END { print add - subs }'

elif [ "$1" = "--all" ]; then
    # 所有提交：输出【累计总行数】（前缀和）
    git log --all --reverse --pretty=tformat:"%H" --numstat -- '*.go' | awk '
    BEGIN { total = 0 }
    NF == 1 {
        if (commit != "") {
            total += (add - subs)
            print total
        }
        commit = $0
        add = 0
        subs = 0
        next
    }
    { add += $1; subs += $2 }
    END {
        total += (add - subs)
        print total
    }'

elif [ "$1" = "--day" ]; then
    # 按天统计：每天最后一次提交后的累计总行数（前缀和）
    git log --all --reverse --date=short --pretty="%cd %H" --numstat -- '*.go' | awk '
    BEGIN { total = 0 }
        {
            if (NF == 2) {
                # 先把上一个提交的变更累加进去
                total += (add - subs)
                current_date = $1
                if (last_date != "" && last_date != current_date) {
                    print last_date " | " total
                }
                last_date = current_date
                add = 0; subs = 0; next
            }
            add += $1; subs += $2
        }
        END {
            total += (add - subs)
            print last_date " | " total
        }'

else
    echo "使用说明："
    echo "  ./xglcc.sh --latest    查看最近一次提交后的累计有效总行数"
    echo "  ./xglcc.sh --all       查看所有提交历史的累计总行数（前缀和）"
    echo "  ./xglcc.sh --day       按天查看代码量趋势（每天的累计总行数）"
    exit 1
fi