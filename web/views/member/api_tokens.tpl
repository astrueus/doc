<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>API Token - {{.site_title_suffix}}</title>
    <link href="{{cdncss "/static/bootstrap/css/bootstrap.min.css"}}" rel="stylesheet">
    <link href="{{cdncss "/static/font-awesome/css/font-awesome.min.css"}}" rel="stylesheet">
    <link href="{{cdncss "/static/css/main.css" "version"}}" rel="stylesheet">
</head>
<body>
<div class="manual-reader">
    {{template "widgets/header.tpl" .}}
    <div class="container manual-body">
        <div class="row">
            <div class="page-left">
                <ul class="menu">
                    <li><a href="{{urlfor "SettingController.Index"}}" class="item"><i class="fa fa-sitemap" aria-hidden="true"></i> {{i18n .Lang "uc.base_info"}}</a></li>
                    <li><a href="{{urlfor "SettingController.Password"}}" class="item"><i class="fa fa-user" aria-hidden="true"></i> {{i18n .Lang "uc.change_pwd"}}</a></li>
                    <li class="active"><a href="{{urlfor "MemberApiTokenController.Index"}}" class="item"><i class="fa fa-key" aria-hidden="true"></i> API Token</a></li>
                </ul>
            </div>
            <div class="page-right">
                <div class="m-box">
                    <div class="box-head">
                        <strong class="box-title">API Token</strong>
                        <button type="button" class="btn btn-success btn-sm pull-right" data-toggle="modal" data-target="#createTokenModal">生成新 Token</button>
                    </div>
                </div>
                <div class="box-body">
                    <p class="text-muted">用于 MCP HTTP 接入。明文只在创建时显示一次，之后仅保存哈希。</p>
                    <table class="table table-hover">
                        <thead>
                        <tr>
                            <th>名称</th>
                            <th>Scopes</th>
                            <th>过期时间</th>
                            <th>最近使用</th>
                            <th>最近 IP</th>
                            <th>状态</th>
                            <th>操作</th>
                        </tr>
                        </thead>
                        <tbody>
                        {{range .Tokens}}
                        <tr>
                            <td>{{.Name}}</td>
                            <td><code>{{.Scopes}}</code></td>
                            <td>{{if .ExpiresAt.IsZero}}永不过期{{else}}{{date_format .ExpiresAt "2006-01-02 15:04"}}{{end}}</td>
                            <td>{{if .LastUsedAt.IsZero}}-{{else}}{{date_format .LastUsedAt "2006-01-02 15:04"}}{{end}}</td>
                            <td>{{if .LastUsedIP}}{{.LastUsedIP}}{{else}}-{{end}}</td>
                            <td>
                                {{if not .RevokedAt.IsZero}}
                                <span class="label label-default">已撤销</span>
                                {{else if and (not .ExpiresAt.IsZero) (.ExpiresAt.Before $.Now)}}
                                <span class="label label-warning">已过期</span>
                                {{else}}
                                <span class="label label-success">有效</span>
                                {{end}}
                            </td>
                            <td>
                                {{if .RevokedAt.IsZero}}
                                <button type="button" class="btn btn-danger btn-xs btn-revoke" data-url="{{urlfor "MemberApiTokenController.Revoke" ":id" .TokenId}}">撤销</button>
                                {{else}}-{{end}}
                            </td>
                        </tr>
                        {{else}}
                        <tr><td colspan="7" class="text-center text-muted">暂无 Token</td></tr>
                        {{end}}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </div>
    {{template "widgets/footer.tpl" .}}
</div>

<div class="modal fade" id="createTokenModal" tabindex="-1" role="dialog">
    <div class="modal-dialog" role="document">
        <div class="modal-content">
            <form id="createTokenForm" method="post" action="{{urlfor "MemberApiTokenController.Create"}}">
                <div class="modal-header">
                    <button type="button" class="close" data-dismiss="modal"><span>&times;</span></button>
                    <h4 class="modal-title">生成新 Token</h4>
                </div>
                <div class="modal-body">
                    <div class="form-group">
                        <label>名称</label>
                        <input type="text" class="form-control" name="name" id="tokenName" maxlength="100" placeholder="例如 Claude Desktop" required>
                    </div>
                    <div class="form-group">
                        <label>Scopes</label>
                        <input type="text" class="form-control" name="scopes" id="tokenScopes" value="read,write" placeholder="read,write">
                    </div>
                    <div class="form-group">
                        <label>过期日期（可空 = 永不过期）</label>
                        <input type="date" class="form-control" name="expires_at" id="tokenExpires">
                    </div>
                    <div id="createError" class="error-message" style="display:none;"></div>
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn btn-default" data-dismiss="modal">取消</button>
                    <button type="submit" class="btn btn-success" data-loading-text="生成中...">生成</button>
                </div>
            </form>
        </div>
    </div>
</div>

<div class="modal fade" id="tokenPlainModal" tabindex="-1" role="dialog" data-backdrop="static">
    <div class="modal-dialog" role="document">
        <div class="modal-content">
            <div class="modal-header">
                <h4 class="modal-title">请立即复制 Token</h4>
            </div>
            <div class="modal-body">
                <div class="alert alert-danger"><strong>关闭窗口后无法再看到明文。</strong>请妥善保存。</div>
                <textarea id="tokenPlainText" class="form-control" rows="3" readonly style="font-family:monospace;"></textarea>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn btn-primary" id="btnCopyToken">复制</button>
                <button type="button" class="btn btn-default" data-dismiss="modal" id="btnClosePlain">我已保存</button>
            </div>
        </div>
    </div>
</div>

<script src="{{cdnjs "/static/jquery/1.12.4/jquery.min.js" "version"}}"></script>
<script src="{{cdnjs "/static/bootstrap/js/bootstrap.min.js" "version"}}"></script>
<script src="{{cdnjs "/static/js/jquery.form.js" "version"}}"></script>
<script src="{{cdnjs "/static/js/main.js" "version"}}"></script>
<script type="text/javascript">
$(function () {
    $("#createTokenForm").ajaxForm({
        beforeSubmit: function () {
            var name = $.trim($("#tokenName").val());
            if (!name) {
                $("#createError").text("请填写名称").show();
                return false;
            }
            $("#createError").hide();
            return true;
        },
        success: function (res) {
            if (res.errcode === 0) {
                $("#createTokenModal").modal("hide");
                $("#tokenPlainText").val(res.data.token);
                $("#tokenPlainModal").modal("show");
                $("#createTokenForm")[0].reset();
                $("#tokenScopes").val("read,write");
            } else {
                $("#createError").text(res.message || "failed").show();
            }
        }
    });

    $("#btnClosePlain").on("click", function () {
        window.location.reload();
    });

    $("#btnCopyToken").on("click", function () {
        var el = document.getElementById("tokenPlainText");
        el.select();
        try { document.execCommand("copy"); } catch (e) {}
    });

    $(document).on("click", ".btn-revoke", function () {
        var url = $(this).data("url");
        if (!confirm("确认撤销该 Token？撤销后不可恢复。")) {
            return;
        }
        $.ajax({
            url: url,
            type: "POST",
            success: function (res) {
                if (res.errcode === 0) {
                    window.location.reload();
                } else {
                    alert(res.message || "revoke failed");
                }
            }
        });
    });
});
</script>
</body>
</html>
