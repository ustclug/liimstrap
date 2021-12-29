(() => {
    domains = [
      "opac.lib.ustc.edu.cn/opac/search",
      "lib.ustc.edu.cn/LIIMS",
      "mail.ustc.edu.cn",
      "email.ustc.edu.cn"
    ]
    for (let i = 0; i < domains.length; i++) {
      if (window.location.href.indexOf(domains[i]) > -1)
        alert("按下「Ctrl + 空格」激活输入法；点击右下角输入法按钮选择其他输入法。")
    }
  })();
  