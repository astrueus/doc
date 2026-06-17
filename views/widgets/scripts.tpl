<!--
  全局脚本注入点（由 BaseController 通过 {{.Scripts}} 注入到所有页面）。

  约束：
  - 本文件不走模板引擎二次渲染（BaseController 直接以 HTML 字符串读取），
    因此不能使用 {{cdnjs}} / {{cdncss}} 等模板函数；统一用 /static/... 相对路径。
  - 若部署在二级路径或独立 CDN，请把此处的资源 URL 改为绝对地址。
  - 浏览器加载顺序：Mermaid → KaTeX → auto-render；都做了 typeof 守卫，缺失不会阻断其它功能。
-->

<script src="/static/editor.md/lib/mermaid/mermaid.min.js"></script>
<script src="/static/katex/katex.min.js"></script>
<script src="/static/katex/contrib/auto-render.min.js"></script>
<script>
(function () {
    if (window.mermaid && typeof mermaid.initialize === 'function') {
        try {
            mermaid.initialize({ startOnLoad: false, securityLevel: 'loose' });
        } catch (e) {
            if (window.console) console.warn('[scripts.tpl] mermaid.initialize failed:', e);
        }
    }

    // 阅读页渲染入口：首屏 + AJAX 切文档后均会调用
    window.renderReadPageMath = function (root) {
        root = root || document.getElementById('page-content');
        if (!root) return;

        if (typeof renderMathInElement === 'function') {
            try {
                renderMathInElement(root, {
                    delimiters: [
                        { left: '$$',  right: '$$',  display: true  },
                        { left: '\\[', right: '\\]', display: true  },
                        { left: '\\(', right: '\\)', display: false },
                        { left: '$',   right: '$',   display: false }
                    ],
                    ignoredTags: ['script', 'noscript', 'style', 'textarea', 'pre', 'code'],
                    throwOnError: false
                });
            } catch (e) {
                if (window.console) console.warn('[scripts.tpl] KaTeX auto-render failed:', e);
            }
        }

        if (window.mermaid && typeof mermaid.run === 'function') {
            var nodes = root.querySelectorAll('.lang-mermaid:not([data-processed]), pre.mermaid:not([data-processed]), .mermaid:not([data-processed])');
            if (nodes && nodes.length) {
                try {
                    mermaid.run({ nodes: nodes }).catch(function (err) {
                        if (window.console) console.warn('[scripts.tpl] mermaid.run rejected:', err);
                    });
                } catch (e) {
                    if (window.console) console.warn('[scripts.tpl] mermaid.run threw:', e);
                }
            }
        }
    };

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', function () {
            window.renderReadPageMath(document.getElementById('page-content'));
        });
    } else {
        window.renderReadPageMath(document.getElementById('page-content'));
    }
})();
</script>
