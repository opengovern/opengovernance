import {
    Accordion,
    AccordionBody,
    AccordionHeader,
    Button,
    Card,
    Flex,
    Icon,
    Select,
    SelectItem,
    Tab,
    TabGroup,
    TabList,
    TabPanel,
    TabPanels,
    Text,
    TextInput,
    Subtitle,
    Title,
} from '@tremor/react'
import {
    ChevronDoubleLeftIcon,
    ChevronDownIcon,
    ChevronUpIcon,
    CloudIcon,
    CommandLineIcon,
    FunnelIcon,
    MagnifyingGlassIcon,
    PlayCircleIcon,
    PlusIcon,
    TagIcon,
} from '@heroicons/react/24/outline'
import { Fragment, useEffect, useMemo, useState } from 'react' // eslint-disable-next-line import/no-extraneous-dependencies
import { highlight, languages } from 'prismjs' // eslint-disable-next-line import/no-extraneous-dependencies
import 'prismjs/components/prism-sql' // eslint-disable-next-line import/no-extraneous-dependencies
import 'prismjs/themes/prism.css'
import Editor from 'react-simple-code-editor'
import {
    IServerSideGetRowsParams,
    RowClickedEvent,
    ValueFormatterParams,
} from 'ag-grid-community'
import {
    CheckCircleIcon,
    ExclamationCircleIcon,
} from '@heroicons/react/24/solid'
import { Transition } from '@headlessui/react'
import { useAtom, useAtomValue } from 'jotai'
import {
    useInventoryApiV1QueryList,
    useInventoryApiV1QueryRunCreate,
    useInventoryApiV2AnalyticsCategoriesList,
    useInventoryApiV2QueryList,
} from '../../../../api/inventory.gen'
import Spinner from '../../../../components/Spinner'
import { getErrorMessage } from '../../../../types/apierror'
import DrawerPanel from '../../../../components/DrawerPanel'
import { RenderObject } from '../../../../components/RenderObject'
import Table, { IColumn } from '../../../../components/Table'

import {
    GithubComKaytuIoKaytuEnginePkgInventoryApiRunQueryResponse,
    Api,
    GithubComKaytuIoKaytuEnginePkgInventoryApiSmartQueryItemV2,
    GithubComKaytuIoKaytuEnginePkgControlApiListV2ResponseItem,
    GithubComKaytuIoKaytuEnginePkgControlApiListV2ResponseItemQuery,
    GithubComKaytuIoKaytuEnginePkgControlApiListV2,
    GithubComKaytuIoKaytuEnginePkgBenchmarkApiListV3ResponseItem,
    GithubComKaytuIoKaytuEnginePkgBenchmarkApiListV3ResponseMetaData,
    GithubComKaytuIoKaytuEnginePkgBenchmarkApiListV3,
} from '../../../../api/api'
import { isDemoAtom, queryAtom, runQueryAtom } from '../../../../store'
import AxiosAPI from '../../../../api/ApiConfig'

import { snakeCaseToLabel } from '../../../../utilities/labelMaker'
import { numberDisplay } from '../../../../utilities/numericDisplay'
import TopHeader from '../../../../components/Layout/Header'
import BenchmarkDetail from './BenchmarkDetail'
import Filter from './Filter'
import KFilter from '../../../../components/Filter'
import { useComplianceApiV3BenchmarkFilters } from '../../../../api/compliance.gen'

export const getTable = (
    headers: string[] | undefined,
    details: any[][] | undefined,
    isDemo: boolean
) => {
    const columns: IColumn<any, any>[] = []
    const rows: any[] = []
    const headerField = headers?.map((value, idx) => {
        if (headers.filter((v) => v === value).length > 1) {
            return `${value}-${idx}`
        }
        return value
    })
    if (headers && headers.length) {
        for (let i = 0; i < headers.length; i += 1) {
            const isHide = headers[i][0] === '_'
            columns.push({
                field: headerField?.at(i),
                headerName: snakeCaseToLabel(headers[i]),
                type: 'string',
                sortable: true,
                hide: isHide,
                resizable: true,
                filter: true,
                width: 170,
                cellRenderer: (param: ValueFormatterParams) => (
                    <span className={isDemo ? 'blur-sm' : ''}>
                        {param.value}
                    </span>
                ),
            })
        }
    }
    if (details && details.length) {
        for (let i = 0; i < details.length; i += 1) {
            const row: any = {}
            for (let j = 0; j < columns.length; j += 1) {
                row[headerField?.at(j) || ''] =
                    typeof details[i][j] === 'string'
                        ? details[i][j]
                        : JSON.stringify(details[i][j])
            }
            rows.push(row)
        }
    }
    const count = rows.length
    return {
        columns,
        rows,
        count,
    }
}

const columns: IColumn<
    GithubComKaytuIoKaytuEnginePkgBenchmarkApiListV3ResponseItem,
    any
>[] = [
    {
        field: 'benchmark.id',
        headerName: 'ID',
        type: 'string',
        sortable: true,
        resizable: false,
    },
    {
        field: 'benchmark.title',
        headerName: 'Title',
        type: 'string',
        sortable: true,
        resizable: false,
    },
    {
        field: 'benchmark.connectors',
        headerName: 'Connector',
        type: 'connector',
        sortable: true,
        resizable: false,
        // cellRenderer: (params: any) =>
        //     params.value?.map((item: string, index: number) => {
        //         return `${item} `
        //     }),
    },
    {
        field: 'benchmark.primary_tables',
        headerName: 'Primary Table',
        type: 'string',
        sortable: true,
        resizable: false,
        cellRenderer: (params: any) => {
            if (params.value && params.value.length > 0) {
                return params.value[0]
            } else {
                return ''
            }
        },
    },
    {
        field: 'benchmark.enabled',
        headerName: 'Enabled',
        type: 'string',
        sortable: true,
        resizable: false,
        cellRenderer: (params: any) => {
            return params.value ? 'True' : 'False'
        },
    },
    {
        field: 'benchmark.number_of_controls',
        headerName: 'Number of Controls',
        type: 'string',
        sortable: true,
        resizable: false,
    },

    // {
    //     field: 'connectors',
    //     headerName: 'Primary Table',
    //     type: 'string',
    //     sortable: true,
    //     resizable: false,
    // },
    // {
    //     type: 'string',
    //     width: 130,
    //     resizable: false,
    //     sortable: false,
    //     cellRenderer: (params: any) => (
    //         <Flex
    //             justifyContent="center"
    //             alignItems="center"
    //             className="h-full"
    //         >
    //             <PlayCircleIcon className="h-5 text-openg-500 mr-1" />
    //             <Text className="text-openg-500">Run query</Text>
    //         </Flex>
    //     ),
    // },
]

export default function AllBenchmarks() {
    const [runQuery, setRunQuery] = useAtom(runQueryAtom)
    const [loading, setLoading] = useState(false)
    const [savedQuery, setSavedQuery] = useAtom(queryAtom)
    const [code, setCode] = useState(savedQuery || '')
    const [selectedIndex, setSelectedIndex] = useState(0)
    const [searchCategory, setSearchCategory] = useState('')
    const [selectedRow, setSelectedRow] =
        useState<GithubComKaytuIoKaytuEnginePkgBenchmarkApiListV3ResponseMetaData>()
    const [openDrawer, setOpenDrawer] = useState(false)
    const [openSlider, setOpenSlider] = useState(false)
    const [openSearch, setOpenSearch] = useState(true)
    const [showEditor, setShowEditor] = useState(true)
    const isDemo = useAtomValue(isDemoAtom)
    const [selectedFilter, setSelectedFilters] = useState<string[]>([])
    const [pageSize, setPageSize] = useState(1000)
    const [autoRun, setAutoRun] = useState(false)
    const [engine, setEngine] = useState('odysseus-sql')
    const [query, setQuery] =
        useState<GithubComKaytuIoKaytuEnginePkgBenchmarkApiListV3>()
    const { response: categories, isLoading: categoryLoading } =
        useInventoryApiV2AnalyticsCategoriesList()
    // const { response: queries, isLoading: queryLoading } =
    //     useInventoryApiV2QueryList({
    //         titleFilter: '',
    //         Cursor: 0,
    //         PerPage:25
    //     })
   const { response: filters, isLoading: filtersLoading } =
       useComplianceApiV3BenchmarkFilters()

    const findFilters = (key: string) => {
        const temp = filters?.tags.filter((item, index) => {
            if (item.Key === key) {
                return item
            }
        })

        if (temp) {
            return temp[0]
        }
        return undefined
    }
    const ssr = () => {
        return {
            getRows: (params: IServerSideGetRowsParams) => {
                // setLoading(true)
                const api = new Api()
                api.instance = AxiosAPI
                const temp = query?.tags
                // @ts-ignore
                if (temp) {
                    Object.keys(temp).map((item, index) => {
                        const filtered = selectedFilter.filter((filter, i) => {
                            if (filter == item) {
                                return filter
                            }
                        })
                        if (!filtered || filtered.length == 0) {
                            // @ts-ignore
                            delete temp[item]
                        }
                    })
                }
                if (temp?.length !== query?.tags?.length) {
                    setQuery(
                        // @ts-ignore
                        (prevSelectedItem) => ({
                            ...prevSelectedItem,
                            tags: temp,
                        })
                    )
                }
                let body = {
                    list_of_tables: query?.list_of_tables,
                    primary_table: query?.primary_table,
                    tags: temp,
                    cursor: params.request.startRow
                        ? Math.floor(params.request.startRow / 25)
                        : 0,
                    per_page: 25,
                    root: true,
                    finding_summary: false,
                }
               
                api.compliance
                    .apiV3BenchmarkList(body)
                    .then((resp) => {
                        params.success({
                            rowData: resp.data.items || [],
                            rowCount: resp.data.total_count,
                        })
                        setLoading(false)
                    })
                    .catch((err) => {
                        setLoading(false)

                        console.log(err)
                        params.fail()
                    })
            },
        }
    }

    const serverSideRows = ssr()

    return (
        <>
            <TopHeader />
            {categoryLoading || loading ? (
                <Spinner className="mt-56" />
            ) : (
                <Flex alignItems="start">
                    {/* <DrawerPanel
                        open={openDrawer}
                        onClose={() => setOpenDrawer(false)}
                    >
                        <RenderObject obj={selectedRow} />
                    </DrawerPanel>
                    {openSearch ? (
                        <Card className="sticky w-fit">
                            <TextInput
                                className="w-56 mb-6"
                                icon={MagnifyingGlassIcon}
                                placeholder="Search..."
                                value={searchCategory}
                                onChange={(e) =>
                                    setSearchCategory(e.target.value)
                                }
                            />
                            {recordToArray(
                                categories?.categoryResourceType
                            ).map(
                                (cat) =>
                                    !!cat.resource_types?.filter((catt) =>
                                        catt
                                            .toLowerCase()
                                            .includes(
                                                searchCategory.toLowerCase()
                                            )
                                    ).length && (
                                        <Accordion className="w-56 border-0 rounded-none bg-transparent mb-1">
                                            <AccordionHeader className="pl-0 pr-0.5 py-1 w-full bg-transparent">
                                                <Text className="text-gray-800">
                                                    {cat.value}
                                                </Text>
                                            </AccordionHeader>
                                            <AccordionBody className="p-0 w-full pr-0.5 cursor-default bg-transparent">
                                                <Flex
                                                    flexDirection="col"
                                                    justifyContent="start"
                                                >
                                                    {cat.resource_types
                                                        ?.filter((catt) =>
                                                            catt
                                                                .toLowerCase()
                                                                .includes(
                                                                    searchCategory.toLowerCase()
                                                                )
                                                        )
                                                        .map((subCat) => (
                                                            <Flex
                                                                justifyContent="start"
                                                                onClick={() =>
                                                                    setCode(
                                                                        `select * from kaytu_resources where resource_type = '${subCat}'`
                                                                    )
                                                                }
                                                            >
                                                                <Text className="ml-4 w-full truncate text-start py-2 cursor-pointer hover:text-openg-600">
                                                                    {subCat}
                                                                </Text>
                                                            </Flex>
                                                        ))}
                                                </Flex>
                                            </AccordionBody>
                                        </Accordion>
                                    )
                            )}
                            <Flex justifyContent="end" className="mt-12">
                                <Button
                                    variant="light"
                                    onClick={() => setOpenSearch(false)}
                                >
                                    <ChevronDoubleLeftIcon className="h-4" />
                                </Button>
                            </Flex>
                        </Card>
                    ) : (
                        <Flex
                            flexDirection="col"
                            justifyContent="center"
                            className="min-h-full w-fit"
                        >
                            <Button
                                variant="light"
                                onClick={() => setOpenSearch(true)}
                            >
                                <Flex flexDirection="col" className="gap-4 w-4">
                                    <FunnelIcon />
                                    <Text className="rotate-90">Options</Text>
                                </Flex>
                            </Button>
                        </Flex>
                    )} */}
                    <Flex flexDirection="col" className="w-full ">
                        {/* <Transition.Root show={showEditor} as={Fragment}>
                            <Transition.Child
                                as={Fragment}
                                enter="ease-in-out duration-500"
                                enterFrom="h-0 opacity-0"
                                enterTo="h-fit opacity-100"
                                leave="ease-in-out duration-500"
                                leaveFrom="h-fit opacity-100"
                                leaveTo="h-0 opacity-0"
                            >
                                <Flex flexDirection="col" className="mb-4">
                                    <Card className="relative overflow-hidden">
                                        <Editor
                                            onValueChange={(text) => {
                                                setSavedQuery('')
                                                setCode(text)
                                            }}
                                            highlight={(text) =>
                                                highlight(
                                                    text,
                                                    languages.sql,
                                                    'sql'
                                                )
                                            }
                                            value={code}
                                            className="w-full bg-white dark:bg-gray-900 dark:text-gray-50 font-mono text-sm"
                                            style={{
                                                minHeight: '200px',
                                                // maxHeight: '500px',
                                                overflowY: 'scroll',
                                            }}
                                            placeholder="-- write your SQL query here"
                                        />
                                        {isLoading && isExecuted && (
                                            <Spinner className="bg-white/30 backdrop-blur-sm top-0 left-0 absolute flex justify-center items-center w-full h-full" />
                                        )}
                                    </Card>
                                    <Flex className="w-full mt-4">
                                        <Flex justifyContent="start">
                                            <Text className="mr-2 w-fit">
                                                Maximum rows:
                                            </Text>
                                            <Select
                                                enableClear={false}
                                                className="w-56"
                                                placeholder="1,000"
                                            >
                                                <SelectItem
                                                    value="1000"
                                                    onClick={() =>
                                                        setPageSize(1000)
                                                    }
                                                >
                                                    1,000
                                                </SelectItem>
                                                <SelectItem
                                                    value="3000"
                                                    onClick={() =>
                                                        setPageSize(3000)
                                                    }
                                                >
                                                    3,000
                                                </SelectItem>
                                                <SelectItem
                                                    value="5000"
                                                    onClick={() =>
                                                        setPageSize(5000)
                                                    }
                                                >
                                                    5,000
                                                </SelectItem>
                                                <SelectItem
                                                    value="10000"
                                                    onClick={() =>
                                                        setPageSize(10000)
                                                    }
                                                >
                                                    10,000
                                                </SelectItem>
                                            </Select>
                                            <Text className="mr-2 w-fit">
                                                Engine:
                                            </Text>
                                            <Select
                                                enableClear={false}
                                                className="w-56"
                                                value={engine}
                                            >
                                                <SelectItem
                                                    value="odysseus-sql"
                                                    onClick={() =>
                                                        setEngine(
                                                            'odysseus-sql'
                                                        )
                                                    }
                                                >
                                                    Odysseus SQL
                                                </SelectItem>
                                                <SelectItem
                                                    value="odysseus-rego"
                                                    onClick={() =>
                                                        setEngine(
                                                            'odysseus-rego'
                                                        )
                                                    }
                                                >
                                                    Odysseus Rego
                                                </SelectItem>
                                            </Select>
                                        </Flex>
                                        <Flex className="w-fit gap-x-3">
                                            {!!code.length && (
                                                <Button
                                                    variant="light"
                                                    color="gray"
                                                    icon={CommandLineIcon}
                                                    onClick={() => setCode('')}
                                                >
                                                    Clear editor
                                                </Button>
                                            )}
                                            <Button
                                                icon={PlayCircleIcon}
                                                onClick={() => sendNow()}
                                                disabled={!code.length}
                                                loading={
                                                    isLoading && isExecuted
                                                }
                                                loadingText="Running"
                                            >
                                                Run query
                                            </Button>
                                        </Flex>
                                    </Flex>
                                    <Flex className="w-full">
                                        {!isLoading && isExecuted && error && (
                                            <Flex
                                                justifyContent="start"
                                                className="w-fit"
                                            >
                                                <Icon
                                                    icon={ExclamationCircleIcon}
                                                    color="rose"
                                                />
                                                <Text color="rose">
                                                    {getErrorMessage(error)}
                                                </Text>
                                            </Flex>
                                        )}
                                        {!isLoading &&
                                            isExecuted &&
                                            queryResponse && (
                                                <Flex
                                                    justifyContent="start"
                                                    className="w-fit"
                                                >
                                                    {memoCount === pageSize ? (
                                                        <>
                                                            <Icon
                                                                icon={
                                                                    ExclamationCircleIcon
                                                                }
                                                                color="amber"
                                                                className="ml-0 pl-0"
                                                            />
                                                            <Text color="amber">
                                                                {`Row limit of ${numberDisplay(
                                                                    pageSize,
                                                                    0
                                                                )} reached, results are truncated`}
                                                            </Text>
                                                        </>
                                                    ) : (
                                                        <>
                                                            <Icon
                                                                icon={
                                                                    CheckCircleIcon
                                                                }
                                                                color="emerald"
                                                            />
                                                            <Text color="emerald">
                                                                Success
                                                            </Text>
                                                        </>
                                                    )}
                                                </Flex>
                                            )}
                                    </Flex>
                                </Flex>
                            </Transition.Child>
                        </Transition.Root> */}
                        {/* <Flex flexDirection="row" className="gap-4">
                            <Card
                                onClick={() => {
                                    console.log('salam')
                                }}
                                className="p-3 cursor-pointer dark:ring-gray-500 hover:shadow-md"
                            >
                                <Subtitle className="font-semibold text-gray-800 mb-2">
                                    KPI
                                </Subtitle>
                            </Card>
                            <Card
                                onClick={() => {
                                    console.log('salam')
                                }}
                                className="p-3 cursor-pointer dark:ring-gray-500 hover:shadow-md"
                            >
                                <Subtitle className="font-semibold text-gray-800 mb-2">
                                    KPI
                                </Subtitle>
                            </Card>{' '}
                            <Card
                                onClick={() => {
                                    console.log('salam')
                                }}
                                className="p-3 cursor-pointer dark:ring-gray-500 hover:shadow-md"
                            >
                                <Subtitle className="font-semibold text-gray-800 mb-2">
                                    KPI
                                </Subtitle>
                            </Card>
                        </Flex> */}
                        {/* <Filter
                            type={'findings'}
                            // @ts-ignore
                            onApply={(e) => setQuery(e)}
                        /> */}
                        <Flex
                            flexDirection="row"
                            justifyContent="start"
                            alignItems="center"
                            className="w-full  gap-1  pb-2 flex-wrap"
                            // style={{overflow:"hidden",overflowX:"scroll",overflowY: "hidden"}}
                        >
                            <KFilter
                                // @ts-ignore
                                options={filters?.parent_benchmark_id?.map(
                                    (unique, index) => {
                                        return {
                                            label: unique,
                                            value: unique,
                                        }
                                    }
                                )}
                                type="multi"
                                hasCondition={true}
                                condition={'is'}
                                //@ts-ignore
                                selectedItems={
                                    query?.parent_benchmark_id
                                        ? query?.parent_benchmark_id
                                        : []
                                }
                                onChange={(values: string[]) => {
                                    // @ts-ignore
                                    setQuery(
                                        // @ts-ignore
                                        {
                                            ...query,
                                            parent_benchmark_id: values,
                                        }
                                    )
                                }}
                                label={'Parent Benchmark'}
                                icon={CloudIcon}
                            />
                            <KFilter
                                // @ts-ignore
                                options={filters?.list_of_tables?.map(
                                    (unique, index) => {
                                        return {
                                            label: unique,
                                            value: unique,
                                        }
                                    }
                                )}
                                type="multi"
                                hasCondition={true}
                                condition={'is'}
                                //@ts-ignore
                                selectedItems={
                                    query?.list_of_tables
                                        ? query?.list_of_tables
                                        : []
                                }
                                onChange={(values: string[]) => {
                                    // @ts-ignore
                                    setQuery(
                                        // @ts-ignore
                                        { ...query, list_of_tables: values }
                                    )
                                }}
                                label={'List of Tables'}
                                icon={CloudIcon}
                            />
                            <KFilter
                                // @ts-ignore
                                options={filters?.primary_table?.map(
                                    (unique, index) => {
                                        return {
                                            label: unique,
                                            value: unique,
                                        }
                                    }
                                )}
                                type="multi"
                                hasCondition={true}
                                condition={'is'}
                                //@ts-ignore
                                selectedItems={
                                    query?.primary_table
                                        ? query?.primary_table
                                        : []
                                }
                                onChange={(values: string[]) => {
                                    // @ts-ignore
                                    setQuery(
                                        // @ts-ignore
                                        { ...query, primary_table: values }
                                    )
                                }}
                                label={'Primary Table'}
                                icon={CloudIcon}
                            />
                            <KFilter
                                // @ts-ignore
                                options={filters?.tags?.map((unique, index) => {
                                    return {
                                        label: unique.Key,
                                        value: unique.Key,
                                    }
                                })}
                                type="multi"
                                hasCondition={false}
                                condition={'is'}
                                //@ts-ignore
                                selectedItems={selectedFilter}
                                onChange={(values: string[]) => {
                                    // @ts-ignore
                                    setSelectedFilters(values)
                                }}
                                label={'Add filters'}
                                icon={PlusIcon}
                            />

                            {selectedFilter.map((item, index) => {
                                return (
                                    <div key={index}>
                                        {findFilters(item) &&
                                            findFilters(item)?.Key && (
                                                <>
                                                    <KFilter
                                                        // @ts-ignore
                                                        options={findFilters(
                                                            item
                                                        ).UniqueValues?.map(
                                                            (unique, index) => {
                                                                return {
                                                                    label: unique,
                                                                    value: unique,
                                                                }
                                                            }
                                                        )}
                                                        type="multi"
                                                        hasCondition={true}
                                                        condition={'is'}
                                                        //@ts-ignore
                                                        selectedItems={
                                                            query?.tags &&
                                                            //@ts-ignore
                                                            query?.tags[
                                                                //@ts-ignore
                                                                findFilters(
                                                                    item
                                                                )?.Key
                                                            ]
                                                                ? //@ts-ignore
                                                                  query?.tags[
                                                                      //@ts-ignore
                                                                      findFilters(
                                                                          item
                                                                      )?.Key
                                                                  ]
                                                                : []
                                                        }
                                                        onChange={(
                                                            values: string[]
                                                        ) => {
                                                            // @ts-ignore
                                                            if (
                                                                values.length >
                                                                0
                                                            ) {
                                                                setQuery(
                                                                    // @ts-ignore
                                                                    (
                                                                        prevSelectedItem
                                                                    ) => ({
                                                                        ...prevSelectedItem,
                                                                        tags: {
                                                                            ...prevSelectedItem?.tags,
                                                                            //@ts-ignore
                                                                            [findFilters(
                                                                                item
                                                                            )
                                                                                ?.Key]:
                                                                                values,
                                                                        },
                                                                    })
                                                                )
                                                            }
                                                        }}
                                                        //@ts-ignore
                                                        label={
                                                            //@ts-ignore
                                                            findFilters(item)
                                                                ?.Key
                                                        }
                                                        icon={TagIcon}
                                                    />
                                                </>
                                            )}
                                    </div>
                                )
                            })}
                        </Flex>

                        <Flex className=" mt-2">
                            <Table
                                id="inventory_queries"
                                columns={columns}
                                serverSideDatasource={serverSideRows}
                                loading={loading}
                                onRowClicked={(e) => {
                                    if (e.data) {
                                        setSelectedRow(e?.data?.benchmark)
                                    }
                                    setOpenSlider(true)
                                }}
                                options={{
                                    rowModelType: 'serverSide',
                                    serverSideDatasource: serverSideRows,
                                }}
                            />
                        </Flex>
                    </Flex>
                </Flex>
            )}
            <BenchmarkDetail
                // type="resource"
                selectedItem={selectedRow}
                open={openSlider}
                onClose={() => setOpenSlider(false)}
                onRefresh={() => window.location.reload()}
            />
        </>
    )
}
