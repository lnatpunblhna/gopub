<template>
  <div>

      <div class="panel">
        <panel-title :title="$route.meta.title"></panel-title>
        <div class="panel-body"
             v-loading="load_data"
             element-loading-text="拼命加载中">
          <el-row>
            <el-col :span="20">
              <el-form class="task-create-form" label-width="100px">
                <div class="task-create-section">
                  <h3>选择发布环境</h3>
                  <div class="environment-grid">
                    <button
                      v-for="group in projectGroups"
                      :key="group.level"
                      type="button"
                      class="environment-card"
                      :class="{ active: Number(activeLevel) === Number(group.level) }"
                      @click="select_environment(group.level)">
                      <span>{{ group.label }}</span>
                      <strong>{{ group.options.length }}</strong>
                      <small>可选项目</small>
                    </button>
                  </div>
                  <el-form-item label="项目名称:" label-width="100px">
                    <el-select v-model="pro_id" filterable @change="select_project" :placeholder="project_placeholder" class="task-create-select">
                      <el-option
                        v-for="item in activeOptions"
                        :key="item.value"
                        :label="item.label"
                        :value="item.value">
                        <span class="project-option-name">{{ item.name }}</span>
                        <span v-if="item.lockstatus" class="project-option-status">{{ item.lockstatus }}</span>
                      </el-option>
                    </el-select>
                  </el-form-item>
                </div>
                <el-form-item class="task-create-actions">
                  <el-button type="primary" @click="on_submit_form" :loading="on_submit_loading" :disabled="btn_submit_disable">创建
                  </el-button>
                  <el-button @click="add_lock" :loading="on_submit_loading" :disabled="btn_add_lock_disable">锁定</el-button>
                  <el-button @click="free_lock" :loading="on_submit_loading" :disabled="btn_free_lock_disable">解锁</el-button>
                  
                  <el-button @click="$router.back()">返回</el-button>
                </el-form-item>
              </el-form>
            </el-col>
          </el-row>
        </div>
      </div>
  </div>
</template>
<script type="text/javascript">
  import {panelTitle} from 'components'
  import {port_conf, port_code} from 'common/port_uri'
  import {tools_verify} from 'common/tools'
  import store from 'store'


  export default {
    data() {
      return {
        projects: [],
        activeLevel: 2,
        pro_id: null,
        load_data: false,
        on_submit_loading: false,
        btn_submit_disable: true,
        btn_add_lock_disable: true,
        btn_free_lock_disable: true
      }
    },
    computed: {
      projectGroups() {
        const groupMap = {
          2: {level: 2, label: '预发布环境', options: []},
          3: {level: 3, label: '线上环境', options: []},
          1: {level: 1, label: '测试环境', options: []},
          0: {level: 0, label: '其他环境', options: []}
        }

        this.projects.forEach((project) => {
          const level = Number(project.level || 0)
          const group = groupMap[level] || groupMap[0]
          const lockstatus = this.get_lock_status(project)
          group.options.push({
            label: project.name,
            name: project.name,
            value: project.id,
            lockstatus
          })
        })

        return [groupMap[2], groupMap[3], groupMap[1], groupMap[0]].filter((group) => group.options.length > 0)
      },
      activeGroup() {
        return this.projectGroups.find((group) => Number(group.level) === Number(this.activeLevel)) || this.projectGroups[0] || null
      },
      activeOptions() {
        return this.activeGroup ? this.activeGroup.options : []
      },
      project_placeholder() {
        return this.activeGroup ? `请选择${this.activeGroup.label}项目` : '暂无可选项目'
      }
    },
    created() {
      this.get_project_data()
    },
    methods: {
      get_lock_status(project) {
        const uid = store.state.user_info.user.Id
        if (Number(project.user_lock || 0) <= 0) {
          return ''
        }
        if (Number(project.user_lock) === Number(uid)) {
          return `${project.lockuser || ''}锁定中`
        }
        return '锁定中'
      },
      get_project_level(projectId) {
        const project = this.projects.find((item) => Number(item.id) === Number(projectId))
        return project ? Number(project.level || 0) : null
      },
      ensure_active_environment() {
        if (this.pro_id) {
          const level = this.get_project_level(this.pro_id)
          if (level !== null) {
            this.activeLevel = level
            return
          }
        }
        if (!this.activeGroup && this.projectGroups.length > 0) {
          this.activeLevel = this.projectGroups[0].level
        }
      },
      reset_action_state() {
        this.btn_submit_disable = true
        this.btn_add_lock_disable = true
        this.btn_free_lock_disable = true
      },
      select_environment(level) {
        if (Number(this.activeLevel) === Number(level)) {
          return
        }
        this.activeLevel = level
        this.pro_id = null
        this.reset_action_state()
      },
      //获取数据
      get_project_data() {
        this.load_data = true
        this.$http.get(port_conf.mylist)
          .then(({data: {data}}) => {
            this.projects = data.table_data || []
            this.load_data = false
            if (this.$route.query.id) {
              this.pro_id=this.$route.query.id
              this.ensure_active_environment()
              this.on_submit_form()
              return
            }
            this.ensure_active_environment()
            this.select_project()
          }).catch(() => {

          this.load_data = false
        })
      },
      add_lock(){
        var proId = 0
        
          proId = this.pro_id
        
        if (proId) {
          this.$http.get(port_conf.lock, {
                  params: {
                    projectId:proId,
                    act:1
                  }
              })
              .then(({data: {data}}) => {
              this.$message({
                    message: "锁定成功！",
                    type: 'success'
                })
              this.get_project_data()
          })
        }
      },
      free_lock(){
        var proId = 0
        
        proId = this.pro_id
        
        if (proId) {
          this.$http.get(port_conf.lock, {
                  params: {
                    projectId:proId,
                    act:0
                  }
              })
              .then(({data: {data}}) => {
              this.$message({
                    message: "解锁成功！",
                    type: 'success'
                })
              this.get_project_data()
          })
        }
      },
      select_project(){
        var uid = store.state.user_info.user.Id
        var role = store.state.user_info.user.Role
        var proId = 0
        
        proId = this.pro_id
        this.reset_action_state()
        if(!proId){
          return 
        }

        for (var i in this.projects){
          var project = this.projects[i]
          if(project.id == proId){
            if(project.user_lock > 0){
              if(project.user_lock == uid){
                this.btn_submit_disable=false
                this.btn_add_lock_disable=true
                this.btn_free_lock_disable=false
              }else{
                this.btn_submit_disable=true
                this.btn_add_lock_disable=true
                if(role == 1){
                  this.btn_free_lock_disable=false
                }else{
                  this.btn_free_lock_disable=true
                }
              }
            }else{
              this.btn_submit_disable=false
              this.btn_add_lock_disable=false
              this.btn_free_lock_disable=true
            }
          }
        }
      },
      //提交
      on_submit_form() {
        var proId = 0
        proId = this.pro_id
        
        if (proId) {
          for (var i in  this.projects) {
            var project = this.projects[i]
            if (proId == project.id) {
              console.log(project.repo_type )
              if (project.repo_type == "git") {
                this.$router.push({
                  name: 'taskGit',
                  query: {id: proId}
                })
                return
              }
              if (project.repo_type == "file") {
                this.$router.push({
                  name: 'taskFile',
                  query: {id: proId}
                })
                return
              }
                if (project.repo_type == "jenkins") {
                  this.$router.push({
                    name: 'taskJenkins',
                    query: {id: proId}
                  })
                  return
                }
            }
          }

        }
      }
    },
    components: {
      panelTitle
    }
  }
</script>
<style lang="scss" scoped>
.task-create-form {
  max-width: 980px;
}

.task-create-section {
  padding: 22px 24px 8px;
  border: 1px solid var(--gp-border-soft);
  border-radius: 8px;
  background: #fff;

  h3 {
    margin: 0 0 16px;
    color: var(--gp-text);
    font-size: 16px;
    font-weight: 800;
  }
}

.environment-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
  margin-bottom: 22px;
}

.environment-card {
  display: grid;
  grid-template-columns: 1fr auto;
  grid-template-areas:
    "title count"
    "meta count";
  align-items: center;
  min-height: 74px;
  padding: 14px 16px;
  border: 1px solid var(--gp-border-soft);
  border-radius: 8px;
  background: #f8fafc;
  color: var(--gp-text-muted);
  text-align: left;
  cursor: pointer;
  transition: border-color .2s ease, background .2s ease, box-shadow .2s ease;

  span {
    grid-area: title;
    color: var(--gp-text);
    font-size: 15px;
    font-weight: 800;
  }

  strong {
    grid-area: count;
    color: var(--gp-text);
    font-size: 28px;
    font-weight: 800;
    line-height: 1;
  }

  small {
    grid-area: meta;
    margin-top: 6px;
    font-size: 12px;
  }

  &.active {
    border-color: var(--gp-primary);
    background: var(--gp-primary-soft);
    box-shadow: 0 10px 24px rgba(15, 118, 110, .12);

    span,
    strong {
      color: var(--gp-primary-strong);
    }
  }
}

.task-create-select {
  width: min(520px, 100%);
}

.task-create-actions {
  margin-top: 18px;
}

.project-option-name {
  float: left;
}

.project-option-status {
  float: right;
  color: var(--gp-text-muted);
  font-size: 13px;
}
</style>
