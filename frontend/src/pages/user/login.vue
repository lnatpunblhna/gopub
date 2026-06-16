<template>
    <div class="login-body">
        <section class="login-hero">
            <div class="login-brand">gopub</div>
            <h1>自动发布系统</h1>
            <p>工程发布控制台</p>
            <div class="login-metrics">
                <div>
                    <strong>PRE</strong>
                    <span>预发布</span>
                </div>
                <div>
                    <strong>PROD</strong>
                    <span>生产</span>
                </div>
                <div>
                    <strong>AUDIT</strong>
                    <span>审计</span>
                </div>
            </div>
        </section>
        <div class="loginWarp"
             v-loading="load_data"
             element-loading-text="正在登陆中..."
             @keyup.enter="submit_form">
            <div class="login-title">
                <strong>欢迎回来</strong>
                <span>请输入账号信息</span>
            </div>
            <div class="login-form">
                <el-form ref="form" :model="form" :rules="rules" label-width="0">
                    <el-form-item prop="user_name" class="login-item">
                        <el-input v-model="form.user_name" placeholder="请输入账户名：" class="form-input"
                                  :autofocus="true"></el-input>
                    </el-form-item>
                    <el-form-item prop="user_password" class="login-item">
                        <el-input type="password" v-model="form.user_password" placeholder="请输入账户密码："
                                  class="form-input"></el-input>
                    </el-form-item>
                    <el-form-item class="login-item">
                        <el-button size="large" type="primary" class="form-submit" @click="submit_form">登录</el-button>
                    </el-form-item>
                    <el-form-item class="login-item">
                        <el-button size="large" class="form-query" @click="to_tasklist">上线单查询</el-button>
                    </el-form-item>
                </el-form>
            </div>
        </div>
    </div>
</template>
<script type="text/javascript">
    import {port_user} from 'common/port_uri'
    import {mapActions} from 'vuex'
    import {SET_USER_INFO} from 'store/actions/type'

    export default{
        data(){
            return {
                form: {
                    user_name: '',
                    user_password: ''
                },
                rules: {
                    user_name: [{required: true, message: '请输入账户名！', trigger: 'blur'}],
                    user_password: [{required: true, message: '请输入账户密码！', trigger: 'blur'}]
                },
                //请求时的loading效果
                load_data: false
            }
        },
        methods: {
                ...mapActions({
                    set_user_info: SET_USER_INFO
                }),
        //提交
        submit_form() {
            this.$refs.form.validate((valid) => {
                if (valid) {
                    this.load_data = true
                    //登录提交
                    this.$http.post(port_user.login, this.form)
                            .then(({data: {data, msg}}) => {
                        let isNull = data !== null
                        this.set_user_info({
                        user: data,
                        login: true
                    })
                    this.$message({
                        message: msg,
                        type: 'success'
                    })
                    setTimeout(() => {
                        this.$router.push({path: '/'})
                },
                    500
                )
                })
                .
                    catch(() => {
                        this.load_data = false
                })
                } else {
                    return false
                }
            }
        )
    },
    to_tasklist(){
          this.$router.push({path: '/task/searchlist'})
    }
    }
    }
</script>
<style lang="scss" type="text/css" rel="stylesheet/scss">
    .login-body {
        position: fixed;
        inset: 0;
        display: grid;
        grid-template-columns: minmax(420px, 1fr) 420px;
        gap: 56px;
        align-items: center;
        justify-content: center;
        padding: 56px 8vw;
        width: 100%;
        height: 100%;
        min-height: 680px;
        background-color: #101827;
        background-image: linear-gradient(120deg, rgba(16,24,39,.92), rgba(16,24,39,.74)), url(./images/login_bg.jpg);
        background-repeat: no-repeat;
        background-position: center;
        background-size: cover;

        .login-hero {
            max-width: 720px;
            color: #fff;

            .login-brand {
                display: inline-flex;
                align-items: center;
                height: 38px;
                padding: 0 14px;
                margin-bottom: 26px;
                border: 1px solid rgba(255,255,255,.18);
                border-radius: 8px;
                background: rgba(255,255,255,.08);
                font-size: 14px;
                font-weight: 700;
                letter-spacing: .08em;
                text-transform: uppercase;
            }

            h1 {
                max-width: 620px;
                margin-bottom: 18px;
                font-size: 48px;
                line-height: 1.08;
                font-weight: 800;
                letter-spacing: 0;
            }

            p {
                max-width: 520px;
                color: rgba(255,255,255,.74);
                font-size: 16px;
                line-height: 1.8;
            }

            .login-metrics {
                display: grid;
                grid-template-columns: repeat(3, minmax(0, 1fr));
                gap: 12px;
                margin-top: 36px;
                max-width: 620px;

                div {
                    padding: 16px;
                    border: 1px solid rgba(255,255,255,.14);
                    border-radius: 8px;
                    background: rgba(255,255,255,.08);
                }

                strong {
                    display: block;
                    margin-bottom: 6px;
                    font-size: 18px;
                }

                span {
                    color: rgba(255,255,255,.66);
                    font-size: 12px;
                    line-height: 1.5;
                }
            }
        }

        .loginWarp {
            width: 420px;
            padding: 34px;
            margin: 0;
            background: rgba(255,255,255,.96);
            border: 1px solid rgba(255,255,255,.66);
            border-radius: 8px;
            box-shadow: 0 28px 80px rgba(0,0,0,.24);
            backdrop-filter: blur(18px);

            .login-title {
                display: flex;
                flex-direction: column;
                gap: 8px;
                margin-bottom: 28px;
                text-align: left;

                strong {
                    color: var(--gp-text);
                    font-size: 24px;
                    line-height: 1.2;
                }

                span {
                    color: var(--gp-text-muted);
                    font-size: 13px;
                }
            }

            .login-item {

                .el-input__inner {
                    margin: 0 !important;
                }

            }
            .form-input {

                input {
                    margin-bottom: 8px;
                    font-size: 14px;
                    height: 44px;
                    border: 1px solid var(--gp-border);
                    background: #fff;
                    border-radius: 8px;
                    color: var(--gp-text);
                }

            }
            .form-submit {
                width: 100%;
                color: #fff;
                border-color: var(--gp-primary);
                background: var(--gp-primary);
                border-radius: 8px;

                &:active,
                &:hover {
                    border-color: var(--gp-primary-strong);
                    background: var(--gp-primary-strong);
                }

            }

            .form-query {
                width: 100%;
                color: var(--gp-primary-strong);
                border-color: var(--gp-border);
                background: #fff;
                border-radius: 8px;
            }
        }
    }

    @media (max-width: 900px) {
        .login-body {
            position: relative;
            min-height: 100vh;
            height: auto;
            grid-template-columns: 1fr;
            gap: 28px;
            padding: 32px 18px;

            .login-hero {
                max-width: 420px;
                margin: 0 auto;
                text-align: left;

                .login-brand {
                    margin-bottom: 18px;
                }

                h1 {
                    font-size: 34px;
                }

                .login-metrics {
                    grid-template-columns: repeat(3, minmax(0, 1fr));
                    margin-top: 24px;
                }
            }

            .loginWarp {
                width: min(100%, 420px);
                margin: 0 auto;
                padding: 28px;
            }
        }
    }

    @media (max-width: 520px) {
        .login-body {
            .login-hero {
                .login-metrics {
                    grid-template-columns: 1fr;
                }
            }

            .loginWarp {
                padding: 24px;
            }
        }
    }
</style>
