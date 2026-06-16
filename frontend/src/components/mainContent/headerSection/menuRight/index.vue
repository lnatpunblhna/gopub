<template>
    <div class="menu-right" v-if="get_user_info.login">
        <el-dropdown
                trigger="click"
                placement="bottom-end"
                popper-class="user-dropdown-popper"
                class="user-menu"
                @command="handle_command">
            <button class="user-menu-trigger" type="button" aria-label="用户菜单">
                <span class="user-avatar" v-text="avatar_text"></span>
                <span class="user-meta">
                    <strong v-text="display_name"></strong>
                    <small v-text="role_label"></small>
                </span>
                <i class="fa fa-angle-down"></i>
            </button>
            <template #dropdown>
                <el-dropdown-menu class="user-dropdown-menu">
                    <el-dropdown-item :command="COMMAND_PASSWORD">
                        <i class="icon fa fa-key"></i>
                        <span>修改密码</span>
                    </el-dropdown-item>
                    <el-dropdown-item v-if="is_admin" :command="COMMAND_USERS">
                        <i class="icon fa fa-users"></i>
                        <span>用户管理</span>
                    </el-dropdown-item>
                    <el-dropdown-item divided :command="COMMAND_LOGOUT">
                        <i class="icon fa fa-sign-out"></i>
                        <span>退出登录</span>
                    </el-dropdown-item>
                </el-dropdown-menu>
            </template>
        </el-dropdown>
    </div>
</template>
<script type="text/javascript">
    import {port_user} from 'common/port_uri'
    import {mapGetters, mapActions} from 'vuex'
    import {GET_USER_INFO} from 'store/getters/type'
    import {SET_USER_INFO} from 'store/actions/type'

    const COMMAND_PASSWORD = 'password'
    const COMMAND_USERS = 'users'
    const COMMAND_LOGOUT = 'logout'

    export default{
        data(){
            return {
                COMMAND_PASSWORD,
                COMMAND_USERS,
                COMMAND_LOGOUT
            }
        },
        computed: {
            ...mapGetters({
                get_user_info: GET_USER_INFO
            }),
            user(){
                return this.get_user_info.user || {}
            },
            display_name(){
                return this.user.Realname || this.user.UserName || this.user.Username || this.user.Name || '管理员'
            },
            avatar_text(){
                return String(this.display_name).trim().charAt(0).toUpperCase()
            },
            is_admin(){
                return Number(this.user.Role) === 1
            },
            role_label(){
                return this.is_admin ? '管理员' : '成员'
            }
        },
        methods: {
            ...mapActions({
                set_user_info: SET_USER_INFO
            }),
            handle_command(command){
                if (command === COMMAND_PASSWORD) {
                    this.to_password()
                    return
                }
                if (command === COMMAND_USERS) {
                    this.to_users()
                    return
                }
                if (command === COMMAND_LOGOUT) {
                    this.user_out()
                }
            },
            to_password(){
                const id = this.user.Id || this.user.id || ''
                this.$router.push({path: '/user/changepasswd', query: id ? {id} : {}})
            },
            to_users(){
                this.$router.push({path: '/user/list'})
            },
            user_out(){
                this.$confirm('确定退出当前账号吗？', '退出登录', {
                    confirmButtonText: '确定',
                    cancelButtonText: '取消',
                    type: 'warning'
                }).then(() => {
                    this.$http.post(port_user.logout)
                            .then(({data: {msg = '已退出登录'}}) => {
                                this.finish_logout(msg)
                            })
                            .catch(() => {
                                this.finish_logout('已退出登录')
                            })
                }).catch(() => {})
            },
            finish_logout(message){
                this.$message({
                    message,
                    type: 'success'
                })
                this.set_user_info(null)
                setTimeout(() => {
                    this.$router.replace({name: "login"})
                }, 300)
            }
        }
    }
</script>
