import {
    Accordion,
    AccordionBody,
    AccordionHeader,
    Button,
    Card,
    Flex,
    Grid,
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
} from '@tremor/react'
import {
    ChevronDoubleLeftIcon,
    ChevronDownIcon,
    ChevronUpIcon,
    CommandLineIcon,
    FunnelIcon,
    MagnifyingGlassIcon,
    PlayCircleIcon,
} from '@heroicons/react/24/outline'
import { Fragment, useEffect, useMemo, useState } from 'react' // eslint-disable-next-line import/no-extraneous-dependencies
import { highlight, languages } from 'prismjs' // eslint-disable-next-line import/no-extraneous-dependencies
import 'prismjs/components/prism-sql' // eslint-disable-next-line import/no-extraneous-dependencies
import 'prismjs/themes/prism.css'
import Editor from 'react-simple-code-editor'
import { RowClickedEvent, ValueFormatterParams } from 'ag-grid-community'
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
} from '../../../api/inventory.gen'
import Spinner from '../../../components/Spinner'
import { getErrorMessage } from '../../../types/apierror'
import DrawerPanel from '../../../components/DrawerPanel'
import { RenderObject } from '../../../components/RenderObject'
import Table, { IColumn } from '../../../components/Table'
import {
    GithubComKaytuIoKaytuEnginePkgInventoryApiRunQueryResponse,
    GithubComKaytuIoKaytuEnginePkgInventoryApiSmartQueryItem,
} from '../../../api/api'
import { isDemoAtom, queryAtom, runQueryAtom } from '../../../store'
import { snakeCaseToLabel } from '../../../utilities/labelMaker'
import { numberDisplay } from '../../../utilities/numericDisplay'
import TopHeader from '../../../components/Layout/Header'
import KTable from '@cloudscape-design/components/table'
import {
    Box,
    Header,
    Modal,
    Pagination,
    SpaceBetween,
} from '@cloudscape-design/components'
import AceEditor from 'react-ace-builds'
// import 'ace-builds/src-noconflict/theme-github'
import 'ace-builds/css/ace.css'
import 'ace-builds/css/theme/cloud_editor.css'
import 'ace-builds/css/theme/cloud_editor_dark.css'
import 'ace-builds/css/theme/cloud_editor_dark.css'
import 'ace-builds/css/theme/twilight.css'
import 'ace-builds/css/theme/sqlserver.css'
import 'ace-builds/css/theme/xcode.css'

import CodeEditor from '@cloudscape-design/components/code-editor'
import KButton from '@cloudscape-design/components/button'
export const getTable = (
    headers: string[] | undefined,
    details: any[][] | undefined,
    isDemo: boolean
) => {
    const columns: any[] = []
    const rows: any[] = []
    const column_def: any[] = []
    const headerField = headers?.map((value, idx) => {
        if (headers.filter((v) => v === value).length > 1) {
            return `${value}-${idx}`
        }
        return value
    })
    if (headers && headers.length) {
        for (let i = 0; i < headers.length; i += 1) {
            const isHide = headers[i][0] === '_'
            // columns.push({
            //     field: headerField?.at(i),
            //     headerName: snakeCaseToLabel(headers[i]),
            //     type: 'string',
            //     sortable: true,
            //     hide: isHide,
            //     resizable: true,
            //     filter: true,
            //     width: 170,
            //     cellRenderer: (param: ValueFormatterParams) => (
            //         <span className={isDemo ? 'blur-sm' : ''}>
            //             {param.value}
            //         </span>
            //     ),
            // })
            columns.push({
                id: headerField?.at(i),
                header: snakeCaseToLabel(headers[i]),
                // @ts-ignore
                cell: (item: any) => (
                    <>
                        {/* @ts-ignore */}
                        {typeof item[headerField?.at(i)] == 'string'
                            ? // @ts-ignore
                              item[headerField?.at(i)]
                            : // @ts-ignore
                              JSON.stringify(item[headerField?.at(i)])}
                    </>
                ),
                maxWidth: '200px',
                // sortingField: 'id',
                // isRowHeader: true,
                // maxWidth: 150,
            })
            column_def.push({
                id: headerField?.at(i),
                visible: !isHide,
            })
        }
    }
    if (details && details.length) {
        for (let i = 0; i < details.length; i += 1) {
            const row: any = {}
            for (let j = 0; j < columns.length; j += 1) {
                row[headerField?.at(j) || ''] = details[i][j]
                //     typeof details[i][j] === 'string'
                //         ? details[i][j]
                //         : JSON.stringify(details[i][j])
            }
            rows.push(row)
        }
    }
    const count = rows.length

    return {
        columns,
        column_def,
        rows,
        count,
    }
}

const columns: IColumn<
    GithubComKaytuIoKaytuEnginePkgInventoryApiSmartQueryItem,
    any
>[] = [
    {
        field: 'title',
        headerName: 'Smart queries',
        type: 'string',
        sortable: true,
        resizable: false,
    },
    {
        type: 'string',
        width: 130,
        resizable: false,
        sortable: false,
        cellRenderer: (params: any) => (
            <Flex
                justifyContent="center"
                alignItems="center"
                className="h-full"
            >
                <PlayCircleIcon className="h-5 text-openg-500 mr-1" />
                <Text className="text-openg-500">Run query</Text>
            </Flex>
        ),
    },
]

export default function Query() {
    const [runQuery, setRunQuery] = useAtom(runQueryAtom)
    const [loaded, setLoaded] = useState(false)
    const [savedQuery, setSavedQuery] = useAtom(queryAtom)
    const [code, setCode] = useState(savedQuery || '')
    const [selectedIndex, setSelectedIndex] = useState(0)
    const [searchCategory, setSearchCategory] = useState('')
    const [selectedRow, setSelectedRow] = useState({})
    const [openDrawer, setOpenDrawer] = useState(false)
    const [openSearch, setOpenSearch] = useState(true)
    const [showEditor, setShowEditor] = useState(true)
    const isDemo = useAtomValue(isDemoAtom)
    const [pageSize, setPageSize] = useState(1000)
    const [autoRun, setAutoRun] = useState(false)
    const [engine, setEngine] = useState('cloudql')
    const [page, setPage] = useState(0)

    const [preferences, setPreferences] = useState(undefined)
    const { response: categories, isLoading: categoryLoading } =
        useInventoryApiV2AnalyticsCategoriesList()

    const {
        response: queryResponse,
        isLoading,
        isExecuted,
        sendNow,
        error,
    } = useInventoryApiV1QueryRunCreate(
        {
            page: { no: 1, size: pageSize },
            engine,
            query: code,
        },
        {},
        autoRun
    )

    useEffect(() => {
        if (autoRun) {
            setAutoRun(false)
        }
        if (queryResponse?.query?.length) {
            setSelectedIndex(2)
        } else setSelectedIndex(0)
    }, [queryResponse])

    useEffect(() => {
        if (!loaded && code.length > 0) {
            sendNow()
            setLoaded(true)
        }
    }, [page])

    useEffect(() => {
        if (code.length) setShowEditor(true)
    }, [code])
    const [ace, setAce] = useState()

    useEffect(() => {
        async function loadAce() {
            const ace = await import('ace-builds')
            await import('ace-builds/webpack-resolver')
            ace.config.set('useStrictCSP', true)
            // ace.config.setMode('ace/mode/sql')
            // @ts-ignore
            // ace.edit(element, {
            //     mode: 'ace/mode/sql',
            //     selectionStyle: 'text',
            // })

            return ace
        }

        loadAce()
            .then((ace) => {
                // @ts-ignore
                setAce(ace)
            })
            .finally(() => {})
    }, [])

    useEffect(() => {
        if (runQuery.length > 0) {
            setCode(runQuery)
            setShowEditor(true)
            setRunQuery('')
            setAutoRun(true)
        }
    }, [runQuery])

    const recordToArray = (record?: Record<string, string[]> | undefined) => {
        if (record === undefined) {
            return []
        }

        return Object.keys(record).map((key) => {
            return {
                value: key,
                resource_types: record[key],
            }
        })
    }

    const memoCount = useMemo(
        () =>
            getTable(queryResponse?.headers, queryResponse?.result, isDemo)
                .count,
        [queryResponse, isDemo]
    )

    return (
        <>
            <TopHeader />
            {categoryLoading ? (
                <Spinner className="mt-56" />
            ) : (
                <Flex alignItems="start" flexDirection="col">
                    <Flex
                        flexDirection="row"
                        className="gap-5"
                        justifyContent="start"
                        alignItems="start"
                    >
                        <Modal
                            visible={openDrawer}
                            onDismiss={() => setOpenDrawer(false)}
                            header="Query Result"
                            className="min-w-[500px]"
                            size="large"
                        >
                            <RenderObject obj={selectedRow} />
                        </Modal>
                        {openSearch ? (
                            <Card className="sticky w-fit h-fit max-h-[550px] min-w-max   overflow-y-scroll">
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
                                                                            `select * from og_resources where resource_type = '${subCat}'`
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
                                    <Flex
                                        flexDirection="col"
                                        className="gap-4 w-4"
                                    >
                                        <FunnelIcon />
                                        <Text className="rotate-90">
                                            Options
                                        </Text>
                                    </Flex>
                                </Button>
                            </Flex>
                        )}
                        <CodeEditor
                            ace={ace}
                            language="sql"
                            value={code}
                            languageLabel="SQL"
                            onChange={({ detail }) => {
                                setSavedQuery('')
                                setCode(detail.value)
                            }}
                            preferences={preferences}
                            onPreferencesChange={(e) =>
                                // @ts-ignore
                                setPreferences(e.detail)
                            }
                            loading={isLoading}
                            themes={{
                                light: ['xcode', 'cloud_editor', 'sqlserver'],
                                dark: ['cloud_editor_dark', 'twilight'],
                                // @ts-ignore
                            }}
                        />
                    </Flex>

                    <Flex flexDirection="col" className="w-full ">
                        <Flex flexDirection="col" className="mb-4">
                            {/* <Card className="relative overflow-hidden"> */}
                            {/* <AceEditor
                                            mode="java"
                                            theme="github"
                                            onChange={(text) => {
                                                setSavedQuery('')
                                                setCode(text)
                                            }}
                                            name="editor"
                                            value={code}
                                        /> */}

                            {/* <Editor
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
                                        /> */}
                            {isLoading && isExecuted && (
                                <Spinner className="bg-white/30 backdrop-blur-sm top-0 left-0 absolute flex justify-center items-center w-full h-full" />
                            )}
                            {/* </Card> */}
                            <Flex className="w-full mt-4">
                                <Flex justifyContent="start" className="gap-1">
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
                                            onClick={() => setPageSize(1000)}
                                        >
                                            1,000
                                        </SelectItem>
                                        <SelectItem
                                            value="3000"
                                            onClick={() => setPageSize(3000)}
                                        >
                                            3,000
                                        </SelectItem>
                                        <SelectItem
                                            value="5000"
                                            onClick={() => setPageSize(5000)}
                                        >
                                            5,000
                                        </SelectItem>
                                        <SelectItem
                                            value="10000"
                                            onClick={() => setPageSize(10000)}
                                        >
                                            10,000
                                        </SelectItem>
                                    </Select>
                                    <Text className="mr-2 w-fit">Engine:</Text>
                                    <Select
                                        enableClear={false}
                                        className="w-56"
                                        value={engine}
                                    >
                                        <SelectItem
                                            value="odysseus-sql"
                                            onClick={() =>
                                                setEngine('odysseus-sql')
                                            }
                                        >
                                            CloudQL
                                        </SelectItem>
                                        {/* <SelectItem
                                            value="odysseus-rego"
                                            onClick={() =>
                                                setEngine('odysseus-rego')
                                            }
                                        >
                                            Odysseus Rego
                                        </SelectItem> */}
                                    </Select>
                                </Flex>
                                <Flex className="w-max gap-x-3">
                                    {!!code.length && (
                                        <KButton
                                            className="  w-max min-w-max  "
                                            onClick={() => setCode('')}
                                            iconSvg={
                                                <CommandLineIcon className="w-5 " />
                                            }
                                        >
                                            Clear editor
                                        </KButton>
                                    )}
                                    <KButton
                                        // icon={PlayCircleIcon}
                                        variant="primary"
                                        className="w-max  min-w-[300px]  "
                                        onClick={() => sendNow()}
                                        disabled={!code.length}
                                        loading={isLoading && isExecuted}
                                        loadingText="Running"
                                        iconSvg={
                                            <PlayCircleIcon className="w-5 " />
                                        }
                                    >
                                        Run
                                    </KButton>
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
                                {!isLoading && isExecuted && queryResponse && (
                                    <Flex
                                        justifyContent="start"
                                        className="w-fit"
                                    >
                                        {memoCount === pageSize ? (
                                            <>
                                                <Icon
                                                    icon={ExclamationCircleIcon}
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
                                                    icon={CheckCircleIcon}
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
                        <Grid numItems={1} className="w-full">
                            <KTable
                                className="   min-h-[450px]   "
                                // resizableColumns
                                // variant="full-page"
                                renderAriaLive={({
                                    firstIndex,
                                    lastIndex,
                                    totalItemsCount,
                                }) =>
                                    `Displaying items ${firstIndex} to ${lastIndex} of ${totalItemsCount}`
                                }
                                onSortingChange={(event) => {
                                    // setSort(event.detail.sortingColumn.sortingField)
                                    // setSortOrder(!sortOrder)
                                }}
                                // sortingColumn={sort}
                                // sortingDescending={sortOrder}
                                // sortingDescending={sortOrder == 'desc' ? true : false}
                                // @ts-ignore
                                // stickyHeader={true}
                                resizableColumns={true}
                                // stickyColumns={
                                //  {   first:1,
                                //     last: 1}
                                // }
                                onRowClick={(event) => {
                                    const row = event.detail.item
                                    // @ts-ignore
                                    setSelectedRow(row)
                                    setOpenDrawer(true)
                                }}
                                columnDefinitions={
                                    getTable(
                                        queryResponse?.headers,
                                        queryResponse?.result,
                                        isDemo
                                    ).columns
                                }
                                columnDisplay={
                                    getTable(
                                        queryResponse?.headers,
                                        queryResponse?.result,
                                        isDemo
                                    ).column_def
                                }
                                enableKeyboardNavigation
                                // @ts-ignore
                                items={getTable(
                                    queryResponse?.headers,
                                    queryResponse?.result,
                                    isDemo
                                ).rows?.slice(page * 10, (page + 1) * 10)}
                                loading={isLoading}
                                loadingText="Loading resources"
                                // stickyColumns={{ first: 0, last: 1 }}
                                // stripedRows
                                trackBy="id"
                                empty={
                                    <Box
                                        margin={{
                                            vertical: 'xs',
                                        }}
                                        textAlign="center"
                                        color="inherit"
                                    >
                                        <SpaceBetween size="m">
                                            <b>No Results</b>
                                        </SpaceBetween>
                                    </Box>
                                }
                                header={
                                    <Header className="w-full">
                                        Results{' '}
                                        <span className=" font-medium">
                                            ({memoCount})
                                        </span>
                                    </Header>
                                }
                                pagination={
                                    <Pagination
                                        currentPageIndex={page + 1}
                                        pagesCount={Math.ceil(
                                            // @ts-ignore
                                            getTable(
                                                queryResponse?.headers,
                                                queryResponse?.result,
                                                isDemo
                                            ).rows.length / 10
                                        )}
                                        onChange={({ detail }) =>
                                            setPage(detail.currentPageIndex - 1)
                                        }
                                    />
                                }
                            />
                        </Grid>
                        {/* <TabGroup
                            id="tabs"
                            index={selectedIndex}
                            onIndexChange={setSelectedIndex}
                        >
                            <TabList className="mb-3">
                                <Flex>
                                    <Flex className="w-fit">
                                        {/* <Tab
                                            onClick={() => {
                                                setSavedQuery('')
                                            }}
                                        >
                                            Popular queries
                                        </Tab>
                                        <Tab
                                            onClick={() => {
                                                setSavedQuery('')
                                            }}
                                        >
                                            All queries
                                        </Tab> 
                                        <Tab
                                            className={
                                                queryResponse?.query?.length &&
                                                !isLoading
                                                    ? 'flex'
                                                    : 'hidden'
                                            }
                                        >
                                            Result
                                        </Tab>
                                    </Flex>
                                    {/* <Button
                                        variant="light"
                                        onClick={() => {
                                            if (showEditor) {
                                                setShowEditor(false)
                                                setSavedQuery('')
                                                setCode('')
                                            } else setShowEditor(true)
                                        }}
                                        icon={
                                            showEditor
                                                ? ChevronUpIcon
                                                : ChevronDownIcon
                                        }
                                    >
                                        {showEditor
                                            ? 'Close query editor'
                                            : 'Open query editor'}
                                    </Button> 
                                </Flex>
                            </TabList>
                            <TabPanels>
                                {/* <TabPanel>
                                    <Table
                                        id="popular_query_table"
                                        columns={columns}
                                        rowData={queries
                                            ?.filter((q) => q.tags?.popular)
                                            .sort((a, b) => {
                                                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                                // @ts-ignore
                                                if (a.title < b.title) {
                                                    return -1
                                                }
                                                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                                // @ts-ignore
                                                if (a.title > b.title) {
                                                    return 1
                                                }
                                                return 0
                                            })}
                                        loading={queryLoading}
                                        onRowClicked={(e) => {
                                            setCode(
                                                `-- ${e.data?.title}\n\n${e.data?.query}` ||
                                                    ''
                                            )
                                            document
                                                .getElementById(
                                                    'kaytu-container'
                                                )
                                                ?.scrollTo({
                                                    top: 0,
                                                    behavior: 'smooth',
                                                })
                                        }}
                                    />
                                </TabPanel>
                                <TabPanel>
                                    <Table
                                        id="query_table"
                                        columns={columns}
                                        rowData={queries?.sort((a, b) => {
                                            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                            // @ts-ignore
                                            if (a.title < b.title) {
                                                return -1
                                            }
                                            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                            // @ts-ignore
                                            if (a.title > b.title) {
                                                return 1
                                            }
                                            return 0
                                        })}
                                        loading={queryLoading}
                                        onRowClicked={(e) => {
                                            setCode(
                                                `-- ${e.data?.title}\n\n${e.data?.query}` ||
                                                    ''
                                            )
                                            document
                                                .getElementById(
                                                    'kaytu-container'
                                                )
                                                ?.scrollTo({
                                                    top: 0,
                                                    behavior: 'smooth',
                                                })
                                        }}
                                    />
                                </TabPanel> 
                                <TabPanel>
                                    <div className="p-5 ">
                                     

                                        {/* // <Table
                                        //     title="Query results"
                                        //     id="finder_table"
                                        //     columns={memoColumns}
                                        //     rowData={
                                        //         getTable(
                                        //             queryResponse?.headers,
                                        //             queryResponse?.result,
                                        //             isDemo
                                        //         ).rows
                                        //     }
                                        //     downloadable
                                        //     onRowClicked={(
                                        //         event: RowClickedEvent
                                        //     ) => {
                                        //         setSelectedRow(event.data)
                                        //         setOpenDrawer(true)
                                        //     }}
                                        // />
                                    </div>
                                </TabPanel>
                            </TabPanels>
                        </TabGroup> */}
                    </Flex>
                </Flex>
            )}
        </>
    )
}
