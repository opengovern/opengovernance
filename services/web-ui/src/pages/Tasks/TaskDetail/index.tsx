import { useSetAtom } from 'jotai'
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import axios from 'axios'
import { Card, Flex } from '@tremor/react'
import { DocumentTextIcon } from '@heroicons/react/24/outline'
import {
    Cards,
    Grid,
    Link,
    Pagination,
    SpaceBetween,
    Table,
} from '@cloudscape-design/components'
import Spinner from '../../../components/Spinner'
import {
    BreadcrumbGroup,
    ExpandableSection,
} from '@cloudscape-design/components'
import ContentLayout from '@cloudscape-design/components/content-layout'
import Container from '@cloudscape-design/components/container'
import Header from '@cloudscape-design/components/header'
import Button from '@cloudscape-design/components/button'
import Box from '@cloudscape-design/components/box'
import {
    Tab,
    TabGroup,
    TabList,
    TabPanel,
    TabPanels,
    Text,
    Title,
    Switch,
} from '@tremor/react'
export default function TaskDetail() {
    const { id } = useParams()
    const [loading, setLoading] = useState(false)
    const [task, setTask] = useState<any>()
    const [page, setPage] = useState(1)
    const [total, setTotal] = useState(0)
    const [selected, setSelected] = useState<any>()
    const [results, setResults] = useState<any>()
    const getDetail = () => {
        setLoading(true)
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
            .get(`${url}/main/tasks/api/v1/tasks/${id}`, config)
            .then((res) => {
                setLoading(false)
                setTask(res.data)
            })
            .catch((err) => {
                setLoading(false)
            })
    }
    const getRunResult = () => {
        setLoading(true)
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
            .get(`${url}/main/tasks/api/v1/tasks/run/${id}?cursor=${page}&per_page=10`, config)
            .then((res) => {
                setLoading(false)
                if(res.data.items){
                    setResults(res.data.items)
                }
                setTotal(res.data.total_count)
                //  setTask(res.data)
            })
            .catch((err) => {
                setLoading(false)
            })
    }

    const RunTask = () => {
        setLoading(true)
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
        const body ={
            task_id:id
        }

        axios
            .post(`${url}/main/tasks/api/v1/tasks/run/`, body,config)
            .then((res) => {
                setLoading(false)
                //  setTask(res.data)
            })
            .catch((err) => {
                setLoading(false)
            })
    }
    useEffect(() => {
        getDetail()
        getRunResult()
    }, [])

    const truncate = (text: string | undefined) => {
        if (text) {
            return text.length > 600 ? text.substring(0, 600) + '...' : text
        }
    }
    return (
        <>
            {loading ? (
                <Spinner className="mt-56" />
            ) : (
                <>
                    <BreadcrumbGroup
                        onClick={(event) => {
                            // event.preventDefault()
                        }}
                        items={[
                            {
                                text: 'Tasks',
                                href: `/tasks`,
                            },
                            { text: task?.name, href: '#' },
                        ]}
                        ariaLabel="Breadcrumbs"
                    />

                    <Container
                        disableHeaderPaddings
                        disableContentPaddings
                        className="rounded-xl  bg-[#0f2940] p-0 text-white mt-4"
                        header={
                            <Header
                                className={`bg-[#0f2940] p-4 pt-0 rounded-xl   text-white ${
                                    false ? 'rounded-b-none' : ''
                                }`}
                                variant="h2"
                                description=""
                            >
                                <SpaceBetween size="xxxs" direction="vertical">
                                    <Box className="rounded-xl same text-white pt-3 pl-3 pb-0">
                                        <Grid
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
                                                >
                                                    <span className="text-white">
                                                        {task?.name}
                                                    </span>
                                                </Box>
                                                <Box
                                                    variant="p"
                                                    margin={{
                                                        top: 'xxs',
                                                        bottom: 's',
                                                    }}
                                                >
                                                    <div className="group text-white important  relative flex text-wrap justify-start">
                                                        <Text className="test-start w-full text-white ">
                                                            {/* @ts-ignore */}
                                                            {truncate(
                                                                task?.description
                                                            )}
                                                        </Text>
                                                        <Card className="absolute w-full text-wrap z-40 top-0 scale-0 transition-all p-2 group-hover:scale-100">
                                                            <Text>
                                                                {
                                                                    task?.description
                                                                }
                                                            </Text>
                                                        </Card>
                                                    </div>
                                                </Box>
                                            </div>
                                        </Grid>
                                    </Box>
                                    <Flex className="w-max pl-3">
                                        <Button
                                            variant="primary"
                                            onClick={() => {
                                                RunTask()
                                            }}
                                        >
                                            Run
                                        </Button>
                                    </Flex>
                                </SpaceBetween>
                            </Header>
                        }
                    ></Container>

                    <Table
                        className="mt-2"
                        onRowClick={(event) => {
                            const row = event.detail.item
                           
                            setSelected(row)
                        }}
                        columnDefinitions={[
                            {
                                id: 'key',
                                header: 'Key Name',
                                cell: (item: any) => item.key,
                                maxWidth: 150,
                            },

                            {
                                id: 'value',
                                header: 'Value',
                                cell: (item: any) => item.value,
                               
                            },
                            {
                                id: 'controls_count',
                                header: 'Using control count',
                                cell: (item: any) =>
                                    item?.controls_count
                                        ? item?.controls_count
                                        : 0,
                            },

                            {
                                id: 'queries_count',
                                header: 'Using query count',
                                cell: (item: any) =>
                                    item?.queries_count
                                        ? item?.queries_count
                                        : 0,
                            },
                        ]}
                        columnDisplay={[
                            { id: 'key', visible: true },
                            { id: 'value', visible: true },
                            { id: 'controls_count', visible: true },
                            { id: 'queries_count', visible: true },
                        ]}
                        loading={loading}
                       
                        // @ts-ignore
                        items={results ? results : []}
                        empty={
                            <Box
                                margin={{ vertical: 'xs' }}
                                textAlign="center"
                                color="inherit"
                            >
                                <SpaceBetween size="m">
                                    <b>No resources</b>
                                    {/* <Button>Create resource</Button> */}
                                </SpaceBetween>
                            </Box>
                        }
                        header={
                            <Header
                                actions={
                                    <>
                                        <Button onClick={getRunResult}>
                                            Reload
                                        </Button>
                                    </>
                                }
                                className="w-full"
                            >
                                Results {total != 0 ? `(${total})` : ''}
                            </Header>
                        }
                        pagination={
                            <Pagination
                                currentPageIndex={page}
                                pagesCount={Math.ceil(total / 15)}
                                onChange={({ detail }) =>
                                    setPage(detail.currentPageIndex)
                                }
                            />
                        }
                    />
                </>
            )}
        </>
    )
}
