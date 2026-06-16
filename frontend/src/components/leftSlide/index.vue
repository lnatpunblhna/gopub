<template>
    <div class="left-side">
        <div class="left-side-inner">
            <router-link to="/" class="logo block">
                <div>gopub</div>
            </router-link>
            <el-menu
                    :key="active_menu"
                    class="menu-box"
                    router
                    unique-opened
                    :default-openeds="default_openeds"
                    :default-active="active_menu">
                <el-menu-item
                        class="menu-list"
                        v-for="item in root_menu_data"
                        :key="item.path"
                        :index="item.path">
                    <i class="icon fa" :class="item.icon"></i>
                    <span v-text="item.title" class="text"></span>
                </el-menu-item>
                <el-sub-menu
                        class="menu-group"
                        v-for="item in group_menu_data"
                        :key="item.path"
                        :index="item.path">
                    <template #title>
                        <i class="icon fa" :class="item.icon"></i>
                        <span v-text="item.title" class="text"></span>
                    </template>
                    <el-menu-item
                            class="menu-list"
                            v-for="sub_item in item.child"
                            :index="sub_item.path"
                            :key="sub_item.path">
                        <i class="icon fa" :class="sub_item.icon"></i>
                        <span v-text="sub_item.title" class="text"></span>
                    </el-menu-item>
                </el-sub-menu>
            </el-menu>
        </div>
    </div>
</template>
<script type="text/javascript">
  import store from 'store'

  const MENU_DATA = [{
    title: "主页",
    path: "/home",
    icon: "fa-home"
  }, {
    title: "项目",
    path: "/conf",
    icon: "fa-cubes",
    adminOnly: true,
    child: [{
      title: "项目配置",
      path: "/conf/list",
      icon: "fa-list"
    }, {
      title: "新增项目",
      path: "/conf/add",
      icon: "fa-plus"
    }]
  }, {
    title: "上线单",
    path: "/task",
    icon: "fa-table",
    child: [{
      title: "全部上线单",
      path: "/task/list",
      icon: "fa-list-alt",
      adminOnly: true
    }, {
      title: "我的上线单",
      path: "/task/mylist",
      icon: "fa-user"
    }, {
      title: "创建上线单",
      path: "/task/create",
      icon: "fa-plus-circle"
    }]
  }, {
    title: "运维工具",
    path: "/ops",
    icon: "fa-wrench",
    adminOnly: true,
    child: [{
      title: "Agent 状态",
      path: "/p2p/check",
      icon: "fa-desktop"
    }, {
      title: "预发布统计",
      path: "/other/noauto",
      icon: "fa-bar-chart"
    }, {
      title: "刷新版本号",
      path: "/other/flush",
      icon: "fa-refresh"
    }, {
      title: "Git 版本查看",
      path: "/other/gitpull",
      icon: "fa-code"
    }]
  }, {
    title: "用户",
    path: "/user",
    icon: "fa-users",
    adminOnly: true,
    child: [{
      title: "用户列表",
      path: "/user/list",
      icon: "fa-address-book"
    }, {
      title: "新增用户",
      path: "/user/register",
      icon: "fa-user-plus"
    }]
  }]

  const ACTIVE_MENU_MAP = [{
    pattern: /^\/conf\/(update|detection)\//,
    active: "/conf/list"
  }, {
    pattern: /^\/task\/release\//,
    active: "/task/list"
  }, {
    pattern: /^\/task\/searchrelease\//,
    active: "/task/list"
  }, {
    pattern: /^\/user\/changepasswd/,
    active: "/user/list"
  }]

  export default{
    computed: {
      user_role(){
        return store.state.user_info.user && store.state.user_info.user.Role
      },
      is_admin(){
        return Number(this.user_role) === 1
      },
      nav_menu_data(){
        return this.filter_menu(MENU_DATA)
      },
      root_menu_data(){
        return this.nav_menu_data.filter((item) => !item.child)
      },
      group_menu_data(){
        return this.nav_menu_data.filter((item) => item.child && item.child.length)
      },
      active_menu(){
        const routePath = this.$route.path
        const match = ACTIVE_MENU_MAP.find((item) => item.pattern.test(routePath))
        return match ? match.active : routePath
      },
      default_openeds(){
        const active = this.active_menu
        return this.group_menu_data
                .filter((item) => item.child.some((sub_item) => sub_item.path === active || active.indexOf(sub_item.path + "/") === 0))
                .map((item) => item.path)
      }
    },
    methods: {
      filter_menu(list){
        return list
                .filter((item) => !item.adminOnly || this.is_admin)
                .map((item) => {
                  if (!item.child) {
                    return item
                  }
                  return {
                    ...item,
                    child: item.child.filter((child) => !child.adminOnly || this.is_admin)
                  }
                })
                .filter((item) => !item.child || item.child.length)
      }
    }
  }
</script>
