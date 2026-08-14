/**
 * Created by zzmhot on 2017/3/23.
 *
 * 组件
 *
 * @author: zzmhot
 * @github: https://github.com/zzmhot
 * @email: zzmhot@163.com
 * @Date: 2017/3/23 18:41
 * @Copyright(©) 2017 by zzmhot.
 *
 */

// leftSlide 只由 router/index.js 动态导入,不在此处静态汇总,
// 否则会被并入主 chunk 而失去按需加载。
import mainContent from 'components/mainContent/index.vue'
import panelTitle from 'components/panelTitle/index.vue'
import simpleImageUpload from 'components/simpleImageUpload/index.vue'
import bottomToolBar from 'components/bottomToolBar/index.vue'
import search from 'components/search/index.vue'
import terminal from 'components/terminal/index.vue'
export {
    mainContent,
    panelTitle,
    simpleImageUpload,
    bottomToolBar,
    search,
    terminal
}
