<template>

    <div class="panel">
        <panel-title :title="$route.meta.title">
        
          <router-link style="float: left; margin-right: 10px" :to="{name: 'register'}" tag="span">
            <el-button type="primary" icon="plus" size="small">添加</el-button>
          </router-link>
          <el-button type="warning" size="small" @click="sync_ldap()">从ldap同步</el-button>
          <el-button type="warning" size="small" @click="test_ldap()">测试ldap</el-button>
        </panel-title>

        <div class="panel-body">
            <el-table
                    class="data-table"
                    :data="table_data"
                    v-loading="load_data"
                    element-loading-text="拼命加载中"
                    border
                    style="width: 100%;">
                <el-table-column
                        prop="id"
                        label="id"
                        width="72">
                </el-table-column>
                <el-table-column
                        prop="realname"
                        label="名称"
                        min-width="100"
                >
                </el-table-column>
                   <el-table-column
                        prop="updated_at"
                        label="上线时间"
                        min-width="150"
                        show-overflow-tooltip>
                </el-table-column>
              <el-table-column
                prop="email"
                label="邮箱"
                min-width="180"
                show-overflow-tooltip>
              </el-table-column>
              <el-table-column
                prop="username"
                label="用户名"
                min-width="120"
                show-overflow-tooltip>
              </el-table-column>
                <el-table-column
                        prop="created_at"
                        label="创建时间"
                        min-width="150"
                        show-overflow-tooltip>
                </el-table-column>
                <el-table-column
                        prop="role"
                        label="角色"
                        min-width="130">
                  <template #default="props">
                    <span>{{ getRole(props.row.role) }}</span>
                  </template>
                </el-table-column>
                <el-table-column
                        prop="ldap"
                        label="用户来源"
                        min-width="110">
                  <template #default="props">
                    <span>{{ isLdap(props.row.from_ldap) }}</span>
                  </template>
                </el-table-column>
                <el-table-column
                        label="操作"
                        fixed="right"
                        width="260">
                    <template #default="props">
                        <div class="table-actions">
                        <router-link :to="{name: 'register', query:  {id: props.row.id}}"
                                     tag="span">
                            <el-button size="small" icon="edit">修改</el-button>
                        </router-link>
                        <router-link :to="{name: 'changepasswd', query:  {id: props.row.id}}"
                                     tag="span" v-if="props.row.from_ldap==0">
                            <el-button size="small" icon="edit">设置密码</el-button>
                        </router-link>
                        <el-button type="danger" size="small" icon="delete"
                                   @click="delete_data(props.row.id)" v-if="props.row.from_ldap==0">删除
                        </el-button>
                        </div>
                    </template>
                </el-table-column>
            </el-table>
        </div>
    </div>
</template>
<script type="text/javascript">
    import {panelTitle, bottomToolBar, search} from 'components'
    import {port_user} from 'common/port_uri'
    import store from 'store'
    export default{
        data(){
            return {
                table_data: null,
                load_data: true,
            }
        },
        components: {
            panelTitle,
            bottomToolBar,
            search
        },
        created(){
            this.get_table_data()
        },
        methods: {
            getRole(value) {
                if (value == 1) {
                    return "管理员"
                } else if (value == 10) {
                    return "全部预发布用户"
                } else if (value == 20) {
                    return "单个项目用户"
                }
                return value
            },
            isLdap(value) {
                if (value == 1) {
                    return "ldap用户"
                }
                return "gopub用户"
            },
            submit_search(value) {
                this.select_info = value
                this.$message({
                    message: value,
                    type: 'success'
                })
                this.get_table_data()
            },
            //刷新
            on_refresh(){
                this.select_info = ""
                this.get_table_data()
            },
            //获取数据
            get_table_data(){
                this.load_data = true
                this.$http.get(port_user.users)
                        .then(({data: {data}}) => {
                    this.table_data = data
                    this.load_data = false
            })
            .
                catch(() => {
                    this.load_data = false
            })
            }

        }
    }
</script>
