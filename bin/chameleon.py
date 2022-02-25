#!/usr/bin/env python3

import re

def main():
    # get profile name by /proc/cmdline
    with open('/proc/cmdline') as f:
        cmdline = f.read()
    profile_re = re.search(r'profile=(\w+)', cmdline)
    if profile_re:
        profile = profile_re.group(1)
        if profile == 'iat':
            print("Profile: iat")
            with open("/home/liims/.config/midori/config") as f:
                midori_config = f.read()
            # replace to iat homepage
            midori_config.replace(
                "http://pxe.ustc.edu.cn/liims",
                "http://pxe.ustc.edu.cn/liims/index_iat.html"
            )
            # replace default search to iat lib book search
            midori_config.replace(
                "http://opac.lib.ustc.edu.cn",
                "http://iat.lib.ustc.edu.cn:88"
            )
        else:
            print("Unknown profile. Do nothing.")

if __name__ == "__main__":
    main()
