<template>

    <div class="panel">
        <panel-title :title="$route.meta.title">
            <div style="float: left;margin-right: 10px">
                <search @search="submit_search"></search>
            </div>
            <el-button @click.stop="on_refresh" size="small">
                <i class="fa fa-refresh"></i>
            </el-button>
        </panel-title>

        <div class="panel-body"  style="clear: both;">
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
                        label="创建人"
                        min-width="96"
                >
                </el-table-column>
                <el-table-column
                        prop="name"
                        label="项目"
                        min-width="130"
                        show-overflow-tooltip
                >
                </el-table-column>
                <el-table-column
                        prop="title"
                        label="上线单标题"
                        min-width="180"
                        show-overflow-tooltip>
                </el-table-column>

                <el-table-column
                        prop="updated_at"
                        label="上线时间"
                        min-width="150"
                        show-overflow-tooltip>
                </el-table-column>
                <el-table-column
                        prop="branch"
                        label="分支"
                        min-width="120"
                        show-overflow-tooltip>
                </el-table-column>
                <el-table-column
                        prop="commit_id"
                        label="上线commit号"
                        min-width="160"
                        show-overflow-tooltip>
                </el-table-column>
                <el-table-column
                        prop="pms_batch_id"
                        label="发布批次ID"
                        min-width="110">
                </el-table-column>
                <el-table-column
                        prop="pms_uwork_id"
                        label="uwork流程号"
                        min-width="110">
                </el-table-column>
                <el-table-column
                        prop="status"
                        label="当前状态"
                        min-width="100">
                </el-table-column>
                <el-table-column
                        label="操作"
                        fixed="right"
                        width="130">
                    <template #default="props">
                        <div class="table-actions">
                        <router-link :to="{name: 'searchtaskRelease', params: {id: props.row.id,is_log:1}}"
                                     v-if="props.row.status!='新建提交'" tag="span">
                            <el-button size="small" icon="edit">查看日志</el-button>
                        </router-link>
                        </div>
                    </template>
                </el-table-column>
            </el-table>
            <bottom-tool-bar>

                <template #page>
                <div>
                    <el-pagination
                            @current-change="handleCurrentChange"
                            :current-page="currentPage"
                            :page-size="15"
                            layout="total, prev, pager, next"
                            :total="total">
                    </el-pagination>
                </div>
                </template>
            </bottom-tool-bar>

        </div>
    </div>
</template>
<script type="text/javascript">
    import {panelTitle, bottomToolBar, search} from 'components'
    import {port_task} from 'common/port_uri'
    export default{
        data(){
            return {
                table_data: null,
                //当前页码
                currentPage: 1,
                //数据总条目
                total: 0,
                //每页显示多少条数据
                length: 15,
                //请求时的loading效果
                load_data: true,
                //批量选择数组
                batch_select: [],
                //批量选择数组
                select_info: ""
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
                console.log(this.currentPage)
                this.load_data = true
                this.$http.get(port_task.list, {
                            params: {
                                page: this.currentPage,
                                length: this.length,
                                select_info: this.select_info
                            }
                        })
                        .then(({data: {data}}) => {
                    this.table_data = data.table_data
                this.currentPage = data.currentPage
                this.total = data.total
                this.load_data = false
            })
            .
                catch(() => {
                    this.load_data = false
            })
            },
    
            //页码选择
            handleCurrentChange(val) {
                this.currentPage = val
                this.get_table_data()
            }
        }
    }
</script>
