<template>
    <div>
        <el-row v-show="isShowStep">
            <el-col :span="24">
                <el-steps :active="stepActive" :status="stepStatus">
                    <el-step title="步骤 1" description="权限、目录检查"></el-step>
                    <el-step title="步骤 2" description="pre-deploy任务"></el-step>
                    <el-step title="步骤 3" description="代码检出"></el-step>
                    <el-step title="步骤 4" description="post-deploy任务"></el-step>
                    <el-step title="步骤 5" description="同步至服务器"></el-step>
                    <el-step title="步骤 6" description="全量更新(pre-release、更新版本、post-release)"></el-step>
                </el-steps>
            </el-col>
        </el-row>
        <el-row v-if="attempts.length > 1" class="attempt-bar">
            <el-col :span="24">
                <span class="attempt-label">历史发布：</span>
                <el-select v-model="attempt" size="small" style="width: 260px" @change="switchAttempt">
                    <el-option v-for="item in attempts"
                               :key="item.value"
                               :label="item.label"
                               :value="item.value"></el-option>
                </el-select>
            </el-col>
        </el-row>
        <div class="terminal-box">
            <div v-for="(line, index) in lines" :key="index">
                <!-- 这里过去用的是 v-html，命令输出里的任何标签都会被当成 HTML 执行；
                     改成插值渲染，交给浏览器转义 -->
                <pre class="terminal-line" :class="line.cls">{{ line.text }}</pre>
                <el-link v-if="line.recordId"
                         type="primary"
                         :underline="false"
                         @click="download(line.recordId)">下载完整日志
                </el-link>
            </div>
            <pre v-if="lines.length === 0" class="terminal-line dim">暂无输出</pre>
        </div>
    </div>

</template>
<script type="text/javascript">
    import {port_record} from 'common/port_uri'

    // 与后端 release.go 的 stage 对应：10/20/.../60 是六个步骤，100 是终结记录
    const ACTION_FINISHED = 100

    export default{
        props: ['taskId', 'isJson'],
        data(){
            return {
                lines: [],
                // 增量拉取的游标：只取 id 大于它的记录，不再每 2 秒把整个任务的日志重取一遍
                lastId: 0,
                maxStage: 0,
                finished: false,
                failed: false,
                timer: null,
                since: Math.floor(Date.now() / 1000) - 10,
                // 发布批次：重新发布不再删除旧日志，0 表示跟随最新一次
                attempt: 0,
                attempts: []
            }
        },
        computed: {
            isShowStep(){
                return this.taskId * 1 > 0
            },
            // active 表示「前几步已完成」，所以进行中的第 N 步取 N-1；结束时六步全亮
            stepActive(){
                if (this.finished) {
                    return this.failed ? Math.max(this.maxStage - 1, 0) : 6
                }
                return Math.max(this.maxStage - 1, 0)
            },
            stepStatus(){
                return this.failed ? 'error' : 'process'
            }
        },
        created(){
            this.loadAttempts()
            this.start()
        },
        beforeUnmount(){
            this.stop()
        },
        methods: {
            start(){
                this.get_data()
                this.stop()
                this.timer = setInterval(this.get_data, 2000)
            },
            stop(){
                if (this.timer) {
                    clearInterval(this.timer)
                    this.timer = null
                }
            },
            clearView(){
                this.lines = []
                this.lastId = 0
                this.maxStage = 0
                this.finished = false
                this.failed = false
            },
            // 重新部署时由父组件调用，attempt 由部署接口返回。
            // 不自己去查「最新批次」是因为此时新一轮的记录可能还没落库，会拿到上一批。
            reset(attempt){
                this.clearView()
                this.attempt = attempt > 0 ? attempt : 0
                this.since = Math.floor(Date.now() / 1000) - 10
                this.loadAttempts()
                this.start()
            },
            switchAttempt(){
                this.clearView()
                this.start()
            },
            // 只有真实上线单才有批次概念，检测 / 刷新那些占位 taskId 不需要
            loadAttempts(){
                if (this.taskId * 1 <= 0) {
                    return
                }
                this.$http.get(port_record.attempts, {
                    params: {taskId: this.taskId}
                }).then(({data: {data}}) => {
                    const rows = data || []
                    this.attempts = rows.map((row) => {
                        const no = parseInt(row.attempt, 10) || 1
                        const failed = String(row.min_status) === '0'
                        return {
                            value: no,
                            label: '第 ' + no + ' 次 · ' + this.formatTime(row.started_at) +
                                ' · ' + (failed ? '有失败' : '正常')
                        }
                    })
                    // 首次进入页面时跟随最新一批
                    if (this.attempt === 0 && this.attempts.length > 0) {
                        this.attempt = this.attempts[0].value
                    }
                }).catch(() => {
                })
            },
            formatTime(ts){
                const n = parseInt(ts, 10)
                if (!n || n <= 0) {
                    return '-'
                }
                const d = new Date(n * 1000)
                const pad = (v) => (v < 10 ? '0' + v : '' + v)
                return pad(d.getMonth() + 1) + '-' + pad(d.getDate()) + ' ' +
                    pad(d.getHours()) + ':' + pad(d.getMinutes())
            },
            get_data(){
                const params = {
                    taskId: this.taskId,
                    lastId: this.lastId,
                    time: this.since
                }
                if (this.attempt > 0) {
                    params.attempt = this.attempt
                }
                this.$http.get(port_record.list, {
                    params: params
                }).then(({data: {data}}) => {
                    const rows = data || []
                    for (let i = 0; i < rows.length; i++) {
                        this.appendRecord(rows[i])
                    }
                    if (this.finished) {
                        this.stop()
                    }
                }).catch(() => {
                })
            },
            appendRecord(row){
                const id = parseInt(row.id, 10) || 0
                if (id > this.lastId) {
                    this.lastId = id
                }
                const action = parseInt(row.action, 10) || 0
                const failed = String(row.status) === '0'
                const stage = this.stageOf(action)
                if (stage > this.maxStage) {
                    this.maxStage = stage
                }
                if (action >= ACTION_FINISHED) {
                    // 后端在流程结束时会写一条终结记录，据此停轮询：
                    // 以前失败了也没有任何标记，页面会一直空转
                    this.finished = true
                    this.failed = failed
                }
                if (failed) {
                    this.failed = true
                }

                this.push(row.command, failed ? 'err' : 'cmd')
                const memo = this.parseMemo(row.memo)
                if (memo && Array.isArray(memo.hosts)) {
                    this.renderStep(memo, id, failed)
                } else if (memo !== null) {
                    this.renderLegacy(memo, failed)
                } else if (row.memo) {
                    this.push(row.memo, failed ? 'err' : 'out')
                }
            },
            // 新格式：后端显式落了 stdout / stderr / errorInfo
            renderStep(memo, recordId, failed){
                for (let i = 0; i < memo.hosts.length; i++) {
                    const h = memo.hosts[i]
                    this.push('--- ' + h.host + ' (exit=' + h.exitCode + ', ' + (h.duration || 0) + 's) ---', 'dim')
                    if (h.stdout) {
                        this.push(h.stdout, failed ? 'err' : 'out')
                    }
                    if (h.stderr) {
                        this.push('stderr:\n' + h.stderr, 'err')
                    }
                    if (h.errorInfo) {
                        this.push('错误:\n' + h.errorInfo, 'err')
                    }
                }
                if (memo.note) {
                    this.push(memo.note, 'dim')
                }
                if (memo.logFile) {
                    this.push(memo.truncated ? '输出已截断' : '完整日志已留存', 'dim', recordId)
                }
            },
            // 旧格式：改造前入库的记录，Error 字段序列化后是空对象，只有 Result 可用
            renderLegacy(memo, failed){
                const cls = failed ? 'err' : 'out'
                const list = Array.isArray(memo) ? memo : [memo]
                for (let i = 0; i < list.length; i++) {
                    const item = list[i]
                    if (!item || typeof item !== 'object') {
                        if (item) {
                            this.push(String(item), cls)
                        }
                        continue
                    }
                    if (item.Host) {
                        this.push('--- ' + item.Host + ' ---', 'dim')
                    }
                    if (item.Result) {
                        this.push(item.Result, cls)
                    }
                    if (item.ErrorInfo) {
                        this.push('错误:\n' + item.ErrorInfo, 'err')
                    }
                }
            },
            parseMemo(memo){
                if (!memo) {
                    return null
                }
                try {
                    return JSON.parse(memo)
                } catch (e) {
                    return null
                }
            },
            push(text, cls, recordId){
                if (text === undefined || text === null || text === '') {
                    return
                }
                this.lines.push({text: String(text), cls: cls || 'out', recordId: recordId || 0})
            },
            stageOf(action){
                if (action >= ACTION_FINISHED) {
                    return 6
                }
                // 新记录写的是 10/20/.../60；兼容早期直接写 1..6 的数据
                if (action >= 10) {
                    return Math.min(Math.floor(action / 10), 6)
                }
                return action
            },
            download(recordId){
                this.$http.get(port_record.log, {
                    params: {recordId: recordId},
                    responseType: 'blob'
                }).then((res) => {
                    const url = window.URL.createObjectURL(new Blob([res.data]))
                    const a = document.createElement('a')
                    a.href = url
                    a.download = 'record-' + recordId + '.log'
                    document.body.appendChild(a)
                    a.click()
                    document.body.removeChild(a)
                    window.URL.revokeObjectURL(url)
                }).catch(() => {
                })
            }
        }
    }
</script>
<style scoped>
    .attempt-bar {
        margin: 5px 5px 0;
    }

    .attempt-label {
        font-size: 12px;
        color: #666;
        margin-right: 4px;
    }

    .terminal-box {
        margin: 5px 5px 0;
        padding: 8px;
        border: 1px dashed rgb(0, 160, 198);
        background-color: #000;
        max-height: 600px;
        overflow-y: auto;
    }

    .terminal-line {
        margin: 0;
        padding: 0;
        background-color: transparent;
        border: none;
        white-space: pre-wrap;
        word-break: break-all;
        font-family: Menlo, Consolas, monospace;
        font-size: 12px;
        line-height: 1.5;
    }

    .terminal-line.cmd {
        color: #00d1ff;
    }

    .terminal-line.out {
        color: #00ff00;
    }

    .terminal-line.err {
        color: #ff5f5f;
    }

    .terminal-line.dim {
        color: #999;
    }
</style>
