<template>
  <div class="home-dashboard">
    <section class="home-hero">
      <div>
        <span class="home-eyebrow">gopub console</span>
        <h2>发布工作台</h2>
        <p>集中查看项目、机器、用户和发布趋势，快速进入常用运维流程。</p>
      </div>
      <div class="home-actions">
        <router-link :to="{ name: 'taskCreate' }">
          <el-button type="primary" icon="plus">创建上线单</el-button>
        </router-link>
        <router-link :to="{ name: 'confList' }">
          <el-button icon="setting">项目配置</el-button>
        </router-link>
      </div>
    </section>

    <section class="home-metrics" aria-label="发布概览">
      <article v-for="item in metricCards" :key="item.key" class="home-metric">
        <span class="home-metric-icon" :class="item.iconClass">
          <i :class="['fa', item.icon]"></i>
        </span>
        <div class="home-metric-body">
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}</strong>
        </div>
      </article>
    </section>

    <section class="home-content-grid">
      <article class="home-card home-card-wide">
        <header class="home-card-header">
          <div>
            <h3>发布趋势</h3>
            <span>按时间窗口统计发布类型占比</span>
          </div>
        </header>
        <div class="home-pie-grid">
          <div v-for="item in periodCards" :key="item.ref" class="home-chart-box">
            <div class="home-chart-title">
              <strong>{{ item.title }}</strong>
              <span>{{ item.total }} 单</span>
            </div>
            <div :ref="item.ref" class="home-chart"></div>
          </div>
        </div>
      </article>

      <article class="home-card">
        <header class="home-card-header">
          <div>
            <h3>快捷入口</h3>
            <span>常用发布与运维页面</span>
          </div>
        </header>
        <div class="home-shortcuts">
          <router-link
            v-for="item in shortcuts"
            :key="item.name"
            :to="{ name: item.name }"
            class="home-shortcut"
          >
            <i :class="['fa', item.icon]"></i>
            <span>{{ item.label }}</span>
          </router-link>
        </div>
      </article>
    </section>

    <section class="home-card">
      <header class="home-card-header">
        <div>
          <h3>项目发布排行</h3>
          <span>按项目聚合最近发布次数</span>
        </div>
      </header>
      <el-tabs v-model="activeName" class="home-tabs" @tab-click="handleClick">
        <el-tab-pane label="本日" name="chartsD">
          <div ref="chartsD" class="home-release-chart"></div>
        </el-tab-pane>
        <el-tab-pane label="本周" name="chartsE">
          <div ref="chartsE" class="home-release-chart"></div>
        </el-tab-pane>
        <el-tab-pane label="本月" name="chartsF">
          <div ref="chartsF" class="home-release-chart"></div>
        </el-tab-pane>
      </el-tabs>
    </section>
  </div>
</template>

<script type="text/javascript">
  import {port_task} from 'common/port_uri'

  export default {
    data() {
      return {
        activeName: 'chartsD',
        echartsInstance: null,
        chartInstances: {},
        periodCards: [
          {ref: 'chartsA', taskType: 'day', title: '本日发布', total: 0},
          {ref: 'chartsB', taskType: 'week', title: '本周发布', total: 0},
          {ref: 'chartsC', taskType: 'month', title: '本月发布', total: 0}
        ],
        totalmem: 0,
        totalpub: 0,
        totalproject: 0,
        totalpubsuccess: 0,
        hostsum: 0,
        shortcuts: [
          {name: 'taskCreate', label: '创建上线单', icon: 'fa-plus-circle'},
          {name: 'taskMyList', label: '我的上线单', icon: 'fa-user'},
          {name: 'confList', label: '项目配置', icon: 'fa-cubes'},
          {name: 'p2pCheck', label: 'Agent 状态', icon: 'fa-desktop'}
        ]
      }
    },
    computed: {
      successRate() {
        return `${this.totalpubsuccess || 0}%`
      },
      metricCards() {
        return [
          {key: 'hosts', label: '服务器', value: this.hostsum || 0, icon: 'fa-desktop', iconClass: 'is-host'},
          {key: 'users', label: '用户', value: this.totalmem || 0, icon: 'fa-users', iconClass: 'is-user'},
          {key: 'projects', label: '项目', value: this.totalproject || 0, icon: 'fa-folder-open', iconClass: 'is-project'},
          {key: 'publish', label: '发布总数', value: this.totalpub || 0, icon: 'fa-cloud-upload', iconClass: 'is-publish'},
          {key: 'rate', label: '成功率', value: this.successRate, icon: 'fa-check-circle', iconClass: 'is-rate'}
        ]
      }
    },
    mounted() {
      this.initDashboard()
      window.addEventListener('resize', this.resizeCharts)
    },
    unmounted() {
      window.removeEventListener('resize', this.resizeCharts)
      Object.values(this.chartInstances).forEach((chart) => chart && chart.dispose())
      this.chartInstances = {}
    },
    methods: {
      async initDashboard() {
        this.echartsInstance = await import('echarts')
        this.$nextTick(() => {
          this.create_totalmen()
          this.periodCards.forEach((item) => this.renderPieChart(item))
          this.create_charts('chartsD', 'dayBypro')
        })
      },
      handleClick(tab) {
        const queryMap = {
          chartsD: 'dayBypro',
          chartsE: 'weekBypro',
          chartsF: 'monthBypro'
        }
        this.create_charts(tab.name, queryMap[tab.name])
      },
      getChart(ref) {
        const dom = this.$refs[ref]
        if (!dom || !this.echartsInstance) return null
        if (!this.chartInstances[ref]) {
          this.chartInstances[ref] = this.echartsInstance.init(dom)
        }
        return this.chartInstances[ref]
      },
      resizeCharts() {
        Object.values(this.chartInstances).forEach((chart) => chart && chart.resize())
      },
      fetchChartData(taskType) {
        return this.$http.get(port_task.chart, {
          params: {taskType}
        }).then(({data: {data}}) => data || [])
      },
      create_totalmen() {
        this.fetchChartData('total').then((data) => {
          this.totalmem = data.totalmen || 0
          this.totalproject = data.totalproject || 0
          this.totalpub = data.totalpub || 0
          this.hostsum = data.hostsum || 0
          const success = data.totalpubsuccess || 0
          this.totalpubsuccess = this.totalpub ? Math.round((success / this.totalpub) * 100) : 0
        }).catch(() => {})
      },
      renderPieChart(item) {
        const chart = this.getChart(item.ref)
        if (!chart) return

        this.fetchChartData(item.taskType).then((data) => {
          let total = 0
          const chartData = data.map((row) => {
            const value = Number(row.task_count || 0)
            total += value
            return {
              value,
              name: row.name
            }
          })
          item.total = total

          chart.setOption({
            color: ['#0f766e', '#2563eb', '#b7791f', '#d64545', '#64748b', '#15803d'],
            tooltip: {
              trigger: 'item',
              formatter: '{b}<br/>发布 {c} 单 ({d}%)'
            },
            legend: {
              bottom: 0,
              icon: 'circle',
              itemWidth: 8,
              itemHeight: 8,
              textStyle: {
                color: '#6b778c'
              }
            },
            series: [{
              name: item.title,
              type: 'pie',
              radius: ['46%', '68%'],
              center: ['50%', '42%'],
              minAngle: 8,
              avoidLabelOverlap: true,
              label: {
                formatter: '{b}',
                color: '#172033'
              },
              data: chartData
            }]
          })
        }).catch(() => {})
      },
      create_charts(ref, query) {
        const chart = this.getChart(ref)
        if (!chart || !query) return

        this.fetchChartData(query).then((data) => {
          const rows = data.map((row) => ({
            name: row.name,
            task_count: Number(row.task_count || 0)
          })).sort((a, b) => a.task_count - b.task_count)

          const names = rows.map((row) => row.name)
          const values = rows.map((row) => row.task_count)
          const dom = this.$refs[ref]
          if (dom) {
            dom.style.height = `${Math.max(360, rows.length * 28 + 120)}px`
          }

          chart.setOption({
            color: ['#0f766e'],
            tooltip: {
              trigger: 'axis',
              axisPointer: {type: 'shadow'}
            },
            grid: {
              left: 8,
              right: 20,
              bottom: 12,
              top: 20,
              containLabel: true
            },
            xAxis: {
              type: 'value',
              splitLine: {
                lineStyle: {
                  color: '#edf1f6'
                }
              }
            },
            yAxis: {
              type: 'category',
              data: names,
              axisTick: {show: false},
              axisLine: {show: false},
              axisLabel: {
                interval: 0,
                color: '#6b778c'
              }
            },
            series: [{
              name: '发布次数',
              type: 'bar',
              barWidth: 14,
              data: values,
              itemStyle: {
                borderRadius: [0, 6, 6, 0]
              }
            }]
          })
          chart.resize()
        }).catch(() => {})
      }
    }
  }
</script>

<style lang="scss" type="text/css" rel="stylesheet/scss">
.home-dashboard {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.home-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  min-height: 132px;
  padding: 24px 26px;
  border: 1px solid var(--gp-border-soft);
  border-radius: var(--gp-radius);
  background: #fff;
  box-shadow: var(--gp-shadow);

  h2 {
    margin: 8px 0 8px;
    color: var(--gp-text);
    font-size: 26px;
    font-weight: 800;
    line-height: 1.2;
  }

  p {
    max-width: 620px;
    color: var(--gp-text-muted);
    font-size: 13px;
    line-height: 1.7;
  }
}

.home-eyebrow {
  color: var(--gp-primary-strong);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: .04em;
  text-transform: uppercase;
}

.home-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;

  a {
    display: inline-flex;
    text-decoration: none;
  }
}

.home-metrics {
  display: grid;
  grid-template-columns: repeat(5, minmax(150px, 1fr));
  gap: 14px;
}

.home-metric {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 96px;
  padding: 18px;
  border: 1px solid var(--gp-border-soft);
  border-radius: var(--gp-radius);
  background: #fff;
  box-shadow: 0 10px 28px rgba(23, 32, 51, .06);

  .home-metric-body span {
    display: block;
    color: var(--gp-text-muted);
    font-size: 12px;
    line-height: 1.4;
  }

  strong {
    display: block;
    margin-top: 4px;
    color: var(--gp-text);
    font-size: 26px;
    font-weight: 800;
    line-height: 1.15;
  }
}

.home-metric .home-metric-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  width: 48px;
  height: 48px;
  border-radius: 8px;
  background: var(--gp-primary-soft);
  color: var(--gp-primary-strong);
  font-size: 22px;
  line-height: 1;

  .fa {
    display: block;
    width: 1em;
    line-height: 1;
    text-align: center;
  }

  &.is-user {
    background: #fff1f1;
    color: #c24141;
  }

  &.is-project {
    background: #edf2ff;
    color: #344f9f;
  }

  &.is-publish {
    background: #eaf5ff;
    color: #1476c8;
  }

  &.is-rate {
    background: #eaf8ef;
    color: #15803d;
  }
}

.home-content-grid {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(280px, .8fr);
  gap: 20px;
  align-items: stretch;
}

.home-card {
  min-width: 0;
  padding: 20px;
  border: 1px solid var(--gp-border-soft);
  border-radius: var(--gp-radius);
  background: #fff;
  box-shadow: var(--gp-shadow);
}

.home-card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;

  h3 {
    margin: 0 0 4px;
    color: var(--gp-text);
    font-size: 16px;
    font-weight: 800;
    line-height: 1.3;
  }

  span {
    color: var(--gp-text-muted);
    font-size: 12px;
    line-height: 1.5;
  }
}

.home-pie-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.home-chart-box {
  min-width: 0;
  padding: 14px;
  border: 1px solid var(--gp-border-soft);
  border-radius: var(--gp-radius);
  background: var(--gp-surface-soft);
}

.home-chart-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-height: 24px;

  strong {
    color: var(--gp-text);
    font-size: 13px;
    font-weight: 800;
  }

  span {
    color: var(--gp-primary-strong);
    font-size: 12px;
    font-weight: 800;
  }
}

.home-chart {
  height: 260px;
}

.home-shortcuts {
  display: grid;
  gap: 10px;
}

.home-shortcut {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 10px 12px;
  border: 1px solid var(--gp-border-soft);
  border-radius: 8px;
  background: var(--gp-surface-soft);
  color: var(--gp-text);
  font-weight: 700;
  text-decoration: none;
  transition: border-color .18s ease, background-color .18s ease, color .18s ease;

  i {
    width: 18px;
    color: var(--gp-primary-strong);
    text-align: center;
  }

  &:hover {
    border-color: var(--gp-primary);
    background: var(--gp-primary-soft);
    color: var(--gp-primary-strong);
  }
}

.home-release-chart {
  min-height: 360px;
}

.home-tabs {
  --el-color-primary: var(--gp-primary);
}

@media (max-width: 1320px) {
  .home-metrics {
    grid-template-columns: repeat(3, minmax(150px, 1fr));
  }

  .home-content-grid,
  .home-pie-grid {
    grid-template-columns: 1fr;
  }

  .home-chart {
    height: 300px;
  }
}

@media (max-width: 720px) {
  .home-hero {
    align-items: flex-start;
    flex-direction: column;
  }

  .home-actions,
  .home-actions a,
  .home-actions .el-button {
    width: 100%;
  }

  .home-metrics {
    grid-template-columns: 1fr;
  }

  .home-card {
    padding: 16px;
  }
}
</style>
