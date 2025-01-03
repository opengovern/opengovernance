// @ts-noCheck
import { useCallback, useEffect, useState } from 'react'
import {
    Button,
    Card,
    Flex,
    Text,
    Switch,
    TextInput,
    Accordion,
    AccordionHeader,
    AccordionBody,
    LineChart,
    Grid,
    Col
} from '@tremor/react'
import Table from '@cloudscape-design/components/table'
import Box from '@cloudscape-design/components/box'
import SpaceBetween from '@cloudscape-design/components/space-between'
import TextFilter from '@cloudscape-design/components/text-filter'
import Header from '@cloudscape-design/components/header'
import Badge from '@cloudscape-design/components/badge'
import KButton from '@cloudscape-design/components/button'
import PropertyFilter from '@cloudscape-design/components/property-filter'

import { useAtomValue, useSetAtom } from 'jotai'
import {
    CommandLineIcon,
    BookOpenIcon,
    CodeBracketIcon,
    Cog8ToothIcon,
    MagnifyingGlassIcon,
    ChevronDownIcon,
    ChevronRightIcon,
    ChevronDoubleLeftIcon,
    FunnelIcon,
    CloudIcon,
} from '@heroicons/react/24/outline'
import {
    GridOptions,
    IAggFuncParams,
    ICellRendererParams,
    RowClickedEvent,
    ValueFormatterParams,
} from 'ag-grid-community'
import { useNavigate } from 'react-router-dom'
import Link from '@cloudscape-design/components/link'
import {
    useInventoryApiV3AllBenchmarksControls,
} from '../../../api/compliance.gen'
import {
    GithubComKaytuIoKaytuEnginePkgComplianceApiControlSummary,
    GithubComKaytuIoKaytuEnginePkgControlApiListV2ResponseItem,
} from '../../../api/api'
import TopHeader from '../../../components/Layout/Header'
import {
    searchAtom,
    useFilterState,
    useURLParam,
    useURLState,
} from '../../../utilities/urlstate'
import { getConnectorIcon } from '../../../components/Cards/ConnectorCard'
import { severityBadge } from '../../Governance/Controls'
import {
    exactPriceDisplay,
    numberDisplay,
} from '../../../utilities/numericDisplay'
import { useInventoryApiV3AllQueryCategory } from '../../../api/inventory.gen'
import { Api } from '../../../api/api'
import AxiosAPI from '../../../api/ApiConfig'
import ButtonDropdown from '@cloudscape-design/components/button-dropdown'
import Pagination from '@cloudscape-design/components/pagination'
import CollectionPreferences from '@cloudscape-design/components/collection-preferences'
import KFilter from '../../../components/Filter'
import {
    BreadcrumbGroup,
    Container,
    ContentLayout,
    DateRangePicker,
    ExpandableSection,
  
} from '@cloudscape-design/components'
import KGrid from '@cloudscape-design/components/grid'
import './style.css'
import Filter from '../../Governance/Compliance/NewBenchmarkSummary/Filter'
import Evaluate from '../../Governance/Compliance/NewBenchmarkSummary/Evaluate'
import axios from 'axios'
import { notificationAtom } from '../../../store'
import ReactEcharts from 'echarts-for-react'
import { numericDisplay } from '../../../utilities/numericDisplay'

interface IRecord
    extends GithubComKaytuIoKaytuEnginePkgComplianceApiControlSummary {
    serviceName: string
    tags: string[]
    passedResourcesCount?: number
}

interface IDetailCellRenderer {
    data: IRecord
}

// const DetailCellRenderer = ({ data }: IDetailCellRenderer) => {
//     const searchParams = useAtomValue(searchAtom)
//     return (
//         <Flex
//             flexDirection="row"
//             className="w-full h-full"
//             alignItems="center"
//             justifyContent="between"
//         >
//             <Text className="ml-12 truncate">{data.control?.description}</Text>
//             <Link
//                 className="mr-2"
//                 to={`${data?.control?.id || ''}?${searchParams}`}
//             >
//                 <Button size="xs">Open</Button>
//             </Link>
//         </Flex>
//     )
// }

export default function ScoreCategory() {
    const { value: selectedConnections } = useFilterState()
    const [category, setCategory] = useURLParam('score_category', '')
    const [listofTables, setListOfTables] = useState([])
    const [selectedBread, setSelectedBread] = useState()
    const [tree, setTree] = useState()

    const [selectedServiceNames, setSelectedServiceNames] = useURLState<
        string[]
    >(
        [],
        (v) => {
            const res = new Map<string, string[]>()
            res.set('serviceNames', v)
            return res
        },
        (v) => {
            return v.get('serviceNames') || []
        }
    )
    const [queries, setQueries] = useState({
        tokens: [],
        operation: 'and',
    })

    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)
    const [hideZero, setHideZero] = useState(true)
    const [quickFilterValue, setQuickFilterValue] = useState<string>('')
    const [openSearch, setOpenSearch] = useState(true)
    const [loading, setLoading] = useState(true)
    const [page, setPage] = useState<number>(0)
    const [rows, setRows] = useState<
        GithubComKaytuIoKaytuEnginePkgControlApiListV2ResponseItem[]
    >([])
    const [totalPage, setTotalPage] = useState<number>(0)
    const [totalCount, setTotalCount] = useState<number>(0)

    const [filters, setFilters] = useState([])
    const [filterOption, setFilterOptions] = useState([])
    const [benchmarkDetail, setBenchmarkDetail] = useState()
    const [sort, setSort] = useState('incidents')
    const [sortOrder, setSortOrder] = useState(true)
    const [selected, setSelected] = useState()
    const [isLoading, setIsLoading] = useState(false)
   const [treePage, setTreePage] = useState(0)
   const [treeTotal, setTreeTotal] = useState(0)
   const [treeTotalPages, setTreeTotalPages] = useState(0)

    const [searchCategory, setSearchCategory] = useState('')
    const categories = [
        'security',
        'cost_optimization',
        'operational_excellence',
        'reliability',
        'performance_efficiency',
    ]

    const navigateToInsightsDetails = (id: string) => {
        navigate(`${id}?${searchParams}`)
    }

    const GetControls = (flag: boolean, id: string | undefined) => {
        setLoading(true)
        const api = new Api()
        api.instance = AxiosAPI
        const benchmarks = category

        let body = {
            list_of_tables: listofTables,
            root_benchmark: flag ? [id] : [`sre_${category.toLowerCase()}`],
            cursor: page,
            severity: query?.severity,
            per_page: 10,
            finding_summary: true,
            sort_by: sort,
            sort_order: sortOrder ? 'desc' : 'asc',
        }
        if (listofTables.length == 0) {
            // @ts-ignore
            delete body['list_of_tables']
        }
        api.compliance
            .apiV2ControlList(body)
            .then((resp) => {
                setTotalPage(Math.ceil(resp.data.total_count / 10))
                setTotalCount(resp.data.total_count)
                if (resp.data.items) {
                    setRows(resp.data.items)
                }
                setLoading(false)
            })
            .catch((err) => {
                setLoading(false)
                setRows([])
            })
    }
    const query = {
        benchmarks: [`sre_${category}`, `sre_${category}`],
    }
    const { response: categoriesAll, isLoading: categoryLoading } =
        // @ts-ignore
        useInventoryApiV3AllBenchmarksControls(query)
    const GetTree = () => {
        let url = ''
        if (window.location.origin === 'http://localhost:3000') {
            url = window.__RUNTIME_CONFIG__.REACT_APP_BASE_URL
        } else {
            url = window.location.origin
        }
        // @ts-ignore
        const token = JSON.parse(localStorage.getItem('openg_auth')).token

        const config = {
            headers: {
                Authorization: `Bearer ${token}`,
            },
        }
        axios
            .get(
                `${url}/main/compliance/api/v3/benchmarks/sre_${category.toLowerCase()}/nested`,
                config
            )
            .then((res) => {
                const temp = []

                if (res.data.children) {
                    // temp.push({
                    //     text: res.data.title,
                    //     href: res.data.id,
                    //     type: 'link',
                    // })
                    res.data.children.map((item) => {
                        let childs = {
                            text: truncate(item.title),
                            href: item.id,
                            type: item.children
                                ? 'expandable-link-group'
                                : 'link',
                        }
                        if (item.children && item.children.length > 0) {
                            let child_item = []
                            item.children.map((sub_item) => {
                                child_item.push({
                                    text: truncate(sub_item.title),
                                    href: sub_item.id,
                                    type: 'link',
                                    parentId: item.id,
                                    parentTitle: item.title,
                                })
                            })
                            childs['items'] = child_item
                        }
                        temp.push(childs)
                    })
                    // setTree(res.data.children)
                } else {
                    // temp.push({
                    //     text: res.data.title,
                    //     href: res.data.id,
                    //     type: 'link',
                    // })
                }
                setTreeTotalPages(Math.ceil(temp.length / 12))
                setTreeTotal(temp.length)
                setTreePage(0)
                setTree(temp)
            })
            .catch((err) => {
                console.log(err)
            })
    }
    useEffect(() => {
        GetControls(false)
        GetTree()
    }, [])
    useEffect(() => {
        if (selected) {
            setPage(0)
            GetControls(true, selected)
        }
    }, [selected])
    useEffect(() => {
        if (selected) {
            GetControls(true, selected)
        } else {
            GetControls(false)
        }
    }, [page])
    useEffect(() => {
        if (selected) {
            GetControls(true, selected)
        } else {
            GetControls(false)
        }
    }, [sort, sortOrder])
    // useEffect(() => {
    //     let temp = {}

    //     queries?.tokenGroups?.map((item) => {
    //         // @ts-ignore
    //         if (temp[item.propertyKey] && temp[item.propertyKey].length > 0) {
    //             temp[item.propertyKey].push(item.value.toLowerCase())
    //         } else {
    //             temp[item.propertyKey] = []
    //             temp[item.propertyKey].push(item.value.toLowerCase())
    //         }
    //     })
    //     setQuery(temp)
    // }, [queries])

    // useEffect(() => {
    //     if (selected) {
    //         GetControls(true, selected)
    //     } else {
    //         GetControls(false)
    //     }
    // }, [query])
    useEffect(() => {
        // @ts-ignore
        const temp = []
        // @ts-ignore
        if (queries) {
            // @ts-ignore
            console.log(queries)
            // @ts-ignore
            queries?.tokens?.map((item) => {
                // @ts-ignore
                temp.push(item.value)
            })
        }
        // @ts-ignore

        setListOfTables(temp)
    }, [queries])
    useEffect(() => {
        // @ts-ignore
        const temp = []
        // @ts-ignore

        const temp_options = []

        categoriesAll?.categories.map((cat) => {
            temp_options.push({
                key: cat?.category.replace(/\s/g, ''),
                operators: ['='],
                propertyLabel: `${cat?.category}`,
                groupValuesLabel: `${cat?.category} values`,
            })
            cat.tables?.map((subcat) => {
                temp.push({
                    propertyKey: cat?.category.replace(/\s/g, ''),
                    value: subcat?.table,
                })
            })
        })
        // @ts-ignore

        setFilterOptions(temp_options)
        // @ts-ignore

        setFilters(temp)
    }, [categoriesAll])
    const today = new Date()
    const lastWeek = new Date(
        today.getFullYear(),
        today.getMonth(),
        today.getDate() - 7
    )
    const setNotification = useSetAtom(notificationAtom)

    const [value, setValue] = useState({
        type: 'absolute',
        startDate: lastWeek.toUTCString(),
        endDate: today.toUTCString(),
    })
    const RunBenchmark = (c: any[]) => {
        
        let url = ''
        if (window.location.origin === 'http://localhost:3000') {
            url = window.__RUNTIME_CONFIG__.REACT_APP_BASE_URL
        } else {
            url = window.location.origin
        }
        const body = {
            integration_info: c.map((c) => {
                return {
                    integration_id: c.value,
                }
            }),
        }
        // @ts-ignore
        const token = JSON.parse(localStorage.getItem('openg_auth')).token

        const config = {
            headers: {
                Authorization: `Bearer ${token}`,
            },
        }
        //    console.log(config)
        axios
            .post(
                `${url}/main/schedule/api/v3/compliance/benchmark/sre_${category}/run`,
                body,
                config
            )
            .then((res) => {
                setNotification({
                    text: `Run is Done You Job id is ${res.data.job_id}`,
                    type: 'success',
                })
            })
            .catch((err) => {
                console.log(err)
            })
    }
    const GetBenchmarks = () => {
        setIsLoading(true)
        let url = ''
        if (window.location.origin === 'http://localhost:3000') {
            url = window.__RUNTIME_CONFIG__.REACT_APP_BASE_URL
        } else {
            url = window.location.origin
        }
        // @ts-ignore
        const token = JSON.parse(localStorage.getItem('openg_auth')).token

        const config = {
            headers: {
                Authorization: `Bearer ${token}`,
            },
        }
        const body = {
            benchmarks: [`sre_${category.toLowerCase()}`],
        }
        axios
            .post(
                `${url}/main/compliance/api/v3/compliance/summary/benchmark`,
                body,
                config
            )
            .then((res) => {
                //  const temp = []
                setIsLoading(false)
                setBenchmarkDetail(res.data[0])
            })
            .catch((err) => {
                setIsLoading(false)

                console.log(err)
            })
    }
    const truncate = (text: string | undefined) => {
        if (text) {
            return text.length > 600 ? text.substring(0, 600) + '...' : text
        }
    }
    const [chart, setChart] = useState()
    const [enable, setEnable] = useState<boolean>(false)

    const GetChart = () => {
        
        let url = ''
        if (window.location.origin === 'http://localhost:3000') {
            url = window.__RUNTIME_CONFIG__.REACT_APP_BASE_URL
        } else {
            url = window.location.origin
        }
        // @ts-ignore
        const token = JSON.parse(localStorage.getItem('openg_auth')).token

        const config = {
            headers: {
                Authorization: `Bearer ${token}`,
            },
        }
        axios
            .post(
                `${url}/main/compliance/api/v3/benchmarks/${`sre_${category.toLowerCase()}`}/trend`,
                {},
                config
            )
            .then((res) => {
                const temp = res.data
                const temp_chart = temp?.datapoints?.map((item) => {
                    if (item.findings_summary) {
                        const temp_data = {
                            date: new Date(item.timestamp)
                                .toLocaleDateString('en-US', {
                                    month: 'short',
                                    day: 'numeric',
                                    hour: 'numeric',
                                    minute: 'numeric',
                                    hour12: !1,
                                })
                                .split(',')
                                .join('\n'),
                            // Total:
                            //     item?.findings_summary?.incidents +
                            //     item?.findings_summary?.non_incidents,
                            Incidents: item.findings_summary?.incidents,
                            'Non-Compliant':
                                item.findings_summary?.non_incidents,
                        }
                        return temp_data
                    }
                })
                const new_chart = temp_chart.filter((item) => {
                    if (item) {
                        return item
                    }
                })
                setChart(new_chart)
            })
            .catch((err) => {
                console.log(err)
            })
    }
    const GetEnabled = () => {
        
        let url = ''
        if (window.location.origin === 'http://localhost:3000') {
            url = window.__RUNTIME_CONFIG__.REACT_APP_BASE_URL
        } else {
            url = window.location.origin
        }
        // @ts-ignore
        const token = JSON.parse(localStorage.getItem('openg_auth')).token

        const config = {
            headers: {
                Authorization: `Bearer ${token}`,
            },
        }
        axios
            .get(
                `${url}/main/compliance/api/v3/benchmark/sre_${category.toLowerCase()}/assignments`,
                config
            )
            .then((res) => {
                   if (
                       res.data.status == 'enabled' ||
                       res.data.status == 'auto-enable'
                   ) {
                       setEnable(true)
                       //    setTab(0)
                   } else {
                       setEnable(false)
                       //    setTab(1)
                   }
                // if (res.data) {
                //     if (res.data.items.length > 0) {
                //         setEnable(true)
                //     } else {
                //         setEnable(false)
                //     }
                // } else {
                //     setEnable(false)
                // }
            })
            .catch((err) => {
                console.log(err)
            })
    }
  const options = () => {
      const confine = true
      const opt = {
          tooltip: {
              confine,
              trigger: 'axis',
              axisPointer: {
                  type: 'line',
                  label: {
                      formatter: (param: any) => {
                          let total = 0
                          if (param.seriesData && param.seriesData.length) {
                              for (
                                  let i = 0;
                                  i < param.seriesData.length;
                                  i += 1
                              ) {
                                  total += param.seriesData[i].data
                              }
                          }

                          return `${param.value} (Total: ${total.toFixed(2)})`
                      },
                      // backgroundColor: '#6a7985',
                  },
              },
              valueFormatter: (value: number | string) => {
                  return numericDisplay(value)
              },
              order: 'valueDesc',
          },
          grid: {
              left: 45,
              right: 0,
              top: 20,
              bottom: 40,
          },
          xAxis: {
              type: 'category',
              data: chart?.map((item) => {
                  return item.date
              }),
          },
          yAxis: {
              type: 'value',
          },
          series: [
              {
                  name: 'Incidents',
                  data: chart?.map((item) => {
                      return item.Incidents
                  }),
                  type: 'line',
              },
              {
                  name: 'Non Compliant',

                  data: chart?.map((item) => {
                      return item['Non-Compliant']
                  }),
                  type: 'line',
              },
          ],
      }
      return opt
  }
    useEffect(() => {
        // @ts-ignore
        GetBenchmarks()
          GetTree()
          GetEnabled()

    }, [])
      useEffect(() => {
          // @ts-ignore
          if(enable){
          GetChart()

          }
      }, [enable])
    return (
        <>
            {/* <TopHeader
            // serviceNames={serviceNames}
            // tags={tags}
            // supportedFilters={[
            //     // 'Environment',
            //     // 'Product',
            //     'Cloud Account',
            //     'Service Name',
            //     'Severity',
            //     'Tag',
            //     'Score Category',
            // ]}
            // initialFilters={[
            //     'Score Category',
            //     'Cloud Account',
            //     // 'Product',
            //     'Tag',
            // ]}
            /> */}
            <Container
                disableHeaderPaddings
                disableContentPaddings
                className="rounded-xl  bg-[#0f2940] p-0 text-white"
                footer={
                    false ? (
                        <ExpandableSection
                            header="Additional settings"
                            variant="footer"
                        >
                            <Flex
                                justifyContent="end"
                                className="bg-white p-4 pt-0 mb-2 w-full gap-3    rounded-xl"
                            >
                                <Filter
                                    type={'accounts'}
                                    onApply={(e) => {
                                        // setAccount(e.connector)
                                    }}
                                    // id={id}
                                />
                                <DateRangePicker
                                    onChange={
                                        ({ detail }) => {}
                                        // setValue(detail.value)
                                    }
                                    value={undefined}
                                    disabled={true}
                                    relativeOptions={[
                                        {
                                            key: 'previous-5-minutes',
                                            amount: 5,
                                            unit: 'minute',
                                            type: 'relative',
                                        },
                                        {
                                            key: 'previous-30-minutes',
                                            amount: 30,
                                            unit: 'minute',
                                            type: 'relative',
                                        },
                                        {
                                            key: 'previous-1-hour',
                                            amount: 1,
                                            unit: 'hour',
                                            type: 'relative',
                                        },
                                        {
                                            key: 'previous-6-hours',
                                            amount: 6,
                                            unit: 'hour',
                                            type: 'relative',
                                        },
                                    ]}
                                    isValidRange={(range) => {
                                        if (range.type === 'absolute') {
                                            const [startDateWithoutTime] =
                                                range.startDate.split('T')
                                            const [endDateWithoutTime] =
                                                range.endDate.split('T')
                                            if (
                                                !startDateWithoutTime ||
                                                !endDateWithoutTime
                                            ) {
                                                return {
                                                    valid: false,
                                                    errorMessage:
                                                        'The selected date range is incomplete. Select a start and end date for the date range.',
                                                }
                                            }
                                            if (
                                                new Date(range.startDate) -
                                                    new Date(range.endDate) >
                                                0
                                            ) {
                                                return {
                                                    valid: false,
                                                    errorMessage:
                                                        'The selected date range is invalid. The start date must be before the end date.',
                                                }
                                            }
                                        }
                                        return { valid: true }
                                    }}
                                    i18nStrings={{}}
                                    absoluteFormat="long-localized"
                                    hideTimeOffset
                                    placeholder="Last 7 days"
                                />
                            </Flex>
                        </ExpandableSection>
                    ) : (
                        ''
                    )
                }
                header={
                    <Header
                        className={`bg-[#0f2940] p-4 rounded-xl ${
                            false ? 'rounded-b-none' : ''
                        }  text-white`}
                        variant="h2"
                        description=""
                        actions={''}
                    >
                        <Box
                            className="rounded-xl same text-white"
                            padding={{ vertical: 'l' }}
                        >
                            <KGrid
                                gridDefinition={[
                                    {
                                        colspan: {
                                            default: 12,
                                            xs: 8,
                                            s: 9,
                                        },
                                    },
                                    {
                                        colspan: {
                                            default: 12,
                                            xs: 4,
                                            s: 3,
                                        },
                                    },
                                ]}
                            >
                                <div>
                                    <Box
                                        variant="h1"
                                        className="text-white important"
                                        color="white"
                                    >
                                        <span className="text-white">
                                            {benchmarkDetail?.benchmark_title}
                                        </span>
                                    </Box>
                                    <Box
                                        variant="p"
                                        color="white"
                                        margin={{
                                            top: 'xxs',
                                            bottom: 's',
                                        }}
                                    >
                                        <div className="group text-white important  relative flex text-wrap justify-start">
                                            <Text className="test-start w-full text-white ">
                                                {/* @ts-ignore */}
                                                {truncate(
                                                    benchmarkDetail?.description
                                                )}
                                            </Text>
                                            <Card className="absolute w-full text-wrap z-40 top-0 scale-0 transition-all p-2 group-hover:scale-100">
                                                <Text>
                                                    {
                                                        benchmarkDetail?.description
                                                    }
                                                </Text>
                                            </Card>
                                        </div>
                                    </Box>
                                    <Box
                                        variant="h1"
                                        className="text-white important"
                                        color="white"
                                    ></Box>
                                    <Flex className="w-max">
                                        <Evaluate
                                            id={`sre_${category.toLowerCase()}`}
                                            benchmarkDetail={benchmarkDetail}
                                            assignmentsCount={0}
                                            onEvaluate={(c) => {
                                                RunBenchmark(c)
                                            }}
                                        />
                                    </Flex>
                                </div>
                            </KGrid>
                        </Box>
                    </Header>
                }
            ></Container>
            <Flex flexDirection="col" className="w-full mt-4">
                {chart && enable && (
                    <>
                        <Flex className="bg-white  w-full border-solid border-2    rounded-xl p-4">
                            {/* <LineChart
                                        className="h-80"
                                        data={chart?.length < 0 ? [] : chart}
                                        index="date"
                                        categories={[
                                            // 'Total',
                                            'Incidents',
                                            'Non Compliant',
                                        ]}
                                        colors={['indigo', 'rose', 'cyan']}
                                        noDataText="No Data to Display"
                                        // valueFormatter={dataFormatter}
                                        yAxisWidth={60}
                                        onValueChange={(v) => console.log(v)}
                                    /> */}
                            <ReactEcharts
                                // echarts={echarts}
                                option={options()}
                                className="w-full"
                                onEvents={() => {}}
                            />
                        </Flex>
                    </>
                )}
                <Grid numItems={12} className="gap-4 w-full">
                    <Col numColSpan={12}>
                        <BreadcrumbGroup
                            onClick={(event) => {
                                event.preventDefault()
                                setSelected(event.detail.href)
                            }}
                            items={selectedBread}
                            ariaLabel="Breadcrumbs"
                        />
                    </Col>
                    {tree && tree.length > 0 && (
                        <Col numColSpan={3}>
                            <Flex
                                className="bg-white  w-full border-solid border-2 h-[550px]    rounded-xl gap-1"
                                flexDirection="col"
                            >
                                <>
                                    <SideNavigation
                                        className="w-full scroll  h-[550px] overflow-scroll p-4 pb-0"
                                        activeHref={selected}
                                        header={{
                                            href: `sre_${category.toLowerCase()}`,
                                            text: controls?.benchmark?.title,
                                        }}
                                        onFollow={(event) => {
                                            event.preventDefault()
                                            setSelected(event.detail.href)
                                            const temp = []

                                            if (event.detail.parentId) {
                                                temp.push({
                                                    text: controls?.benchmark
                                                        ?.title,
                                                    href: id,
                                                })
                                                temp.push({
                                                    text: event.detail
                                                        .parentTitle,
                                                    href: event.detail.parentId,
                                                })
                                                temp.push({
                                                    text: event.detail.text,
                                                    href: event.detail.href,
                                                })
                                            } else {
                                                temp.push({
                                                    text: controls?.benchmark
                                                        ?.title,
                                                    href: id,
                                                })
                                                if (
                                                    event.detail.text !==
                                                    controls?.benchmark?.title
                                                ) {
                                                    temp.push({
                                                        text: event.detail.text,
                                                        href: event.detail.href,
                                                    })
                                                }
                                            }
                                            setSelectedBread(temp)
                                        }}
                                        items={tree?.slice(
                                            treePage * 12,
                                            (treePage + 1) * 12
                                        )}
                                    />
                                </>
                                <Pagination
                                    className="pb-2"
                                    currentPageIndex={treePage + 1}
                                    pagesCount={treeTotalPages}
                                    onChange={({ detail }) =>
                                        setTreePage(detail.currentPageIndex - 1)
                                    }
                                />
                            </Flex>
                        </Col>
                    )}
                    <Col numColSpan={tree && tree.length > 0 ? 9 : 12}>
                        {' '}
                        {filters &&
                            filters.length > 0 &&
                            filterOption &&
                            filterOption.length > 0 && (
                                <>
                                    <Table
                                        className="p-3 w-full"
                                        // resizableColumns
                                        onSortingChange={(event) => {
                                            console.log(event)
                                            setSort(
                                                event.detail.sortingColumn
                                                    .sortingField
                                            )
                                            setSortOrder(!sortOrder)
                                        }}
                                        sortingColumn={sort}
                                        sortingDescending={sortOrder}
                                        renderAriaLive={({
                                            firstIndex,
                                            lastIndex,
                                            totalItemsCount,
                                        }) =>
                                            `Displaying items ${firstIndex} to ${lastIndex} of ${totalItemsCount}`
                                        }
                                        columnDefinitions={[
                                            {
                                                id: 'id',
                                                header: 'ID',
                                                cell: (item) => item.id,
                                                sortingField: 'id',
                                                isRowHeader: true,
                                            },
                                            {
                                                id: 'title',
                                                header: 'Title',
                                                cell: (item) => (
                                                    <Link
                                                        href="#"
                                                        onClick={() => {
                                                            navigateToInsightsDetails(
                                                                item.id
                                                            )
                                                        }}
                                                    >
                                                        {item.title}
                                                    </Link>
                                                ),
                                                sortingField: 'title',
                                                // minWidth: 400,
                                                maxWidth: 200,
                                            },
                                            {
                                                id: 'connector',
                                                header: 'Connector',
                                                cell: (item) => item.connector,
                                            },
                                            {
                                                id: 'query',
                                                header: 'Primary Table',
                                                cell: (item) =>
                                                    item?.query?.primary_table,
                                            },
                                            {
                                                id: 'severity',
                                                header: 'Severity',
                                                sortingField: 'severity',
                                                cell: (item) => (
                                                    <Badge
                                                        // @ts-ignore
                                                        color={`severity-${item.severity}`}
                                                    >
                                                        {item.severity
                                                            .charAt(0)
                                                            .toUpperCase() +
                                                            item.severity.slice(
                                                                1
                                                            )}
                                                    </Badge>
                                                ),
                                                maxWidth: 100,
                                            },
                                            {
                                                id: 'query.parameters',
                                                header: 'Has Parametrs',
                                                cell: (item) => (
                                                    // @ts-ignore
                                                    <>
                                                        {item.query?.parameters
                                                            .length > 0
                                                            ? 'True'
                                                            : 'False'}
                                                    </>
                                                ),
                                            },
                                            {
                                                id: 'incidents',
                                                header: 'Incidents',
                                                sortingField: 'incidents',

                                                cell: (item) => (
                                                    // @ts-ignore
                                                    <>
                                                        {/**@ts-ignore */}
                                                        {item?.findings_summary
                                                            ?.incident_count
                                                            ? item
                                                                  ?.findings_summary
                                                                  ?.incident_count
                                                            : 0}
                                                    </>
                                                ),
                                                // minWidth: 50,
                                                maxWidth: 100,
                                            },
                                            {
                                                id: 'passing_resources',
                                                header: 'Non Incidents ',
                                                cell: (item) => (
                                                    // @ts-ignore
                                                    <>
                                                        {item?.findings_summary
                                                            ?.non_incident_count
                                                            ? item
                                                                  ?.findings_summary
                                                                  ?.non_incident_count
                                                            : 0}
                                                    </>
                                                ),
                                                maxWidth: 100,
                                            },
                                            {
                                                id: 'noncompliant_resources',
                                                header: 'Non-Compliant Resources',
                                                sortingField: 'noncompliant_resources',
                                                cell: (item) => (
                                                    // @ts-ignore
                                                    <>
                                                        {item?.findings_summary
                                                            ?.noncompliant_resources
                                                            ? item
                                                                  ?.findings_summary
                                                                  ?.noncompliant_resources
                                                            : 0}
                                                    </>
                                                ),
                                                maxWidth: 100,
                                            },
                                            {
                                                id: 'waste',
                                                header: 'Waste',
                                                cell: (item) => (
                                                    // @ts-ignore
                                                    <>
                                                        {item?.findings_summary
                                                            ?.cost_optimization
                                                            ? item
                                                                  ?.findings_summary
                                                                  ?.cost_optimization
                                                            : 0}
                                                    </>
                                                ),
                                                maxWidth: 100,
                                            },
                                            {
                                                id: 'action',
                                                header: 'Action',
                                                cell: (item) => (
                                                    // @ts-ignore
                                                    <KButton
                                                        onClick={() => {
                                                            navigateToInsightsDetails(
                                                                item.id
                                                            )
                                                        }}
                                                        variant="inline-link"
                                                        ariaLabel={`Open Detail`}
                                                    >
                                                        Open
                                                    </KButton>
                                                ),
                                            },
                                        ]}
                                        columnDisplay={[
                                            { id: 'id', visible: false },
                                            { id: 'title', visible: true },
                                            {
                                                id: 'connector',
                                                visible: false,
                                            },
                                            { id: 'query', visible: false },
                                            {
                                                id: 'severity',
                                                visible: true,
                                            },
                                            {
                                                id: 'incidents',
                                                visible: false,
                                            },
                                            {
                                                id: 'passing_resources',
                                                visible: false,
                                            },
                                            {
                                                id: 'noncompliant_resources',
                                                visible: enable,
                                            },
                                            {
                                                id: 'waste',
                                                visible:
                                                    category == 'Efficiency' &&
                                                    enable
                                                        ? true
                                                        : false,
                                            },

                                            // { id: 'action', visible: true },
                                        ]}
                                        enableKeyboardNavigation
                                        items={rows}
                                        loading={loading}
                                        loadingText="Loading resources"
                                        // stickyColumns={{ first: 0, last: 1 }}
                                        // stripedRows
                                        trackBy="id"
                                        empty={
                                            <Box
                                                margin={{ vertical: 'xs' }}
                                                textAlign="center"
                                                color="inherit"
                                            >
                                                <SpaceBetween size="m">
                                                    <b>No resources</b>
                                                 
                                                </SpaceBetween>
                                            </Box>
                                        }
                                        filter={
                                            <PropertyFilter
                                                // @ts-ignore
                                                query={queries}
                                                // @ts-ignore
                                                onChange={({ detail }) => {
                                                    // @ts-ignore
                                                    setQueries(detail)
                                                }}
                                                // countText="5 matches"
                                                enableTokenGroups
                                                expandToViewport
                                                filteringAriaLabel="Control Categories"
                                                // @ts-ignore
                                                // filteringOptions={filters}
                                                filteringPlaceholder="Control Categories"
                                                // @ts-ignore
                                                filteringOptions={filters}
                                                filteringProperties={
                                                    filterOption
                                                }
                                                // filteringProperties={
                                                //     filterOption
                                                // }
                                            />
                                        }
                                        header={
                                            <Header className="w-full">
                                                Controls{' '}
                                                <span className=" font-medium">
                                                    ({totalCount})
                                                </span>
                                            </Header>
                                        }
                                        pagination={
                                            <Pagination
                                                currentPageIndex={page}
                                                onChange={({ detail }) =>
                                                    setPage(
                                                        detail.currentPageIndex
                                                    )
                                                }
                                                pagesCount={totalPage}
                                            />
                                        }
                                    />
                                </>
                            )}
                    </Col>
                </Grid>
            </Flex>
        </>
    )
}
