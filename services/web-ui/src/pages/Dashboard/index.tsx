import Cal, { getCalApi } from '@calcom/embed-react'
import { Flex, Grid, Icon, Text, Title } from '@tremor/react'
import { useParams, useSearchParams } from 'react-router-dom'
import { useEffect, useState } from 'react'
import TopHeader from '../../components/Layout/Header'
import { Card, Select, SelectItem } from '@tremor/react'
import Modal from '../../components/Modal'
import table from '../../icons/table.png'
import board from '../../icons/board.png'
import { useNavigate } from 'react-router-dom'
import {
    ChevronRightIcon,
    DocumentTextIcon,
    PlusIcon,
} from '@heroicons/react/24/outline'
import { Badge, Cards, Link } from '@cloudscape-design/components'
const data = [
    {
        id: 1,
        name: 'Cloud Assets & Inventory',
        description:
            'Summarized view of all AWS Accounts & Azure Subscriptions Inventory',
        page: 'infrastructure',
        label: 'Built-in',
        icon: board,
    },
    {
        id: 2,
        name: 'Public Cloud Spend',
        description:
            'Summarized view of all Spend on AWS Accounts & Azure Subscriptions',
        page: 'spend',
        label: 'Built-in',
        icon: board,
    },
    {
        id: 3,
        name: 'Cloud Assets by Accounts',
        description:
            'Asset analytics of all AWS Accounts & Azure Subscriptions',
        page: 'infrastructure-cloud-accounts',
        label: 'Built-in',
        icon: table,
    },
    {
        id: 4,
        name: 'Cloud Assets by Service',
        description:
            'Asset analytics of all Cloud services in AWS Accounts & Azure Subscriptions',
        page: 'infrastructure-metrics',
        label: 'Built-in',
        icon: table,
    },
    {
        id: 5,
        name: 'Cloud Spend by Service',
        description:
            'Spend Details of Cloud Services across all  AWS Accounts & Azure Subscriptions',
        page: 'spend-accounts',
        label: 'Built-in',
        icon: table,
    },
    {
        id: 6,
        name: 'Spend Details by Cloud Accounts',
        description: 'Cloud Spend detail at Cloud Account',
        page: 'spend-metrics',
        label: 'Built-in',
        icon: table,
    },
]
export default function Dashboard() {
    const [searchParams, setSearchParams] = useSearchParams()
    const [open, setOpen] = useState<boolean>(false)
    const workspace = useParams<{ ws: string }>().ws
    const navigate = useNavigate()

    const f = async () => {
        const cal = await getCalApi({ namespace: 'try-enterprise' })
        cal('ui', {
            styles: { branding: { brandColor: '#000000' } },
            hideEventTypeDetails: false,
            layout: 'month_view',
        })
    }

    return (
        <>
            {/* <TopHeader /> */}
            {/* <Flex
                className="bg-white w-[90%] rounded-xl border-solid  border-2 border-gray-200   "
                flexDirection="col"
                justifyContent="center"
                alignItems="center"
            >
                <div className="border-b w-full rounded-xl border-tremor-border bg-tremor-background-muted p-4 dark:border-dark-tremor-border dark:bg-gray-950 sm:p-6 lg:p-8">
                    <header>
                        <h1 className="text-tremor-title font-semibold text-tremor-content-strong dark:text-dark-tremor-content-strong">
                            Dashboards
                        </h1>
                        <p className="text-tremor-default text-tremor-content dark:text-dark-tremor-content">
                            Explore and manage your Dashboards
                        </p>
                        <div className="mt-8 w-full md:flex md:max-w-3xl md:items-stretch md:space-x-4">
                            <Card className="w-full md:w-7/12">
                                <div className="inline-flex items-center justify-center rounded-tremor-small border border-tremor-border p-2 dark:border-dark-tremor-border">
                                    <DocumentTextIcon
                                        className="size-5 text-tremor-content-emphasis dark:text-dark-tremor-content-emphasis"
                                        aria-hidden={true}
                                    />
                                </div>
                                <h3 className="mt-4 text-tremor-default font-medium text-tremor-content-strong dark:text-dark-tremor-content-strong">
                                    <a
                                        href="https://docs.opengovernance.io/"
                                        className="focus:outline-none"
                                        target="_blank"
                                    >
                                      
                                        <span
                                            className="absolute inset-0"
                                            aria-hidden={true}
                                        />
                                        Documentation
                                    </a>
                                </h3>
                                <p className="dark:text-dark-tremor-cont text-tremor-default text-tremor-content">
                                    Learn how to use the included dashboard.
                                </p>
                            </Card>
                        </div>
                    </header>
                </div>
                <div className="w-full">
                    <div className="p-4 sm:p-6 lg:p-8">
                        <main>
                            <div className="flex items-center justify-between">
                                <h2 className="text-tremor-title font-semibold text-tremor-content-strong dark:text-dark-tremor-content-strong">
                                    Available Dashboards
                                </h2>
                                <div className="flex items-center space-x-2">
                                    {/* <Select
                                        placeholder="Sorty by"
                                        enableClear={false}
                                        className="[&>button]:rounded-tremor-small"
                                    >
                                        <SelectItem value="1">Name</SelectItem>
                                        <SelectItem value="2">
                                            Last edited
                                        </SelectItem>
                                        <SelectItem value="3">Size</SelectItem>
                                    </Select> 

                                    <button
                                        type="button"
                                        onClick={() => {
                                            f()
                                            setOpen(true)
                                        }}
                                        className="hidden h-9 items-center gap-1.5 whitespace-nowrap rounded-tremor-small bg-tremor-brand px-3 py-2.5 text-tremor-default font-medium text-tremor-brand-inverted shadow-tremor-input hover:bg-tremor-brand-emphasis dark:bg-dark-tremor-brand dark:text-dark-tremor-brand-inverted dark:shadow-dark-tremor-input dark:hover:bg-dark-tremor-brand-emphasis sm:inline-flex"
                                    >
                                        <PlusIcon
                                            className="-ml-1 size-5 shrink-0"
                                            aria-hidden={true}
                                        />
                                        Create new Dashboard
                                    </button>
                                </div>
                            </div>
                            <div className="mt-4">
                                <Cards
                                    cardsPerRow={[
                                        { cards: 1 },
                                        { minWidth: 500, cards: 2 },
                                    ]}
                                    cardDefinition={{
                                        header: (item) => (
                                            <Link
                                                onClick={(e) => {
                                                    // @ts-ignore
                                                    e.preventDefault()
                                                }}
                                                href={`/dashboard/${item.page}`}
                                                fontSize="heading-m"
                                            >
                                                <Flex
                                                    className="w-100"
                                                    justifyContent="between"
                                                    alignItems="center"
                                                >
                                                    <Flex
                                                        className="w-100 min-w-max"
                                                        justifyContent="start"
                                                    >
                                                        {item.name}
                                                    </Flex>
                                                    <Flex
                                                        justifyContent="end"
                                                        className="gap-2"
                                                    >
                                                        <Badge>
                                                            {item.label}
                                                        </Badge>
                                                    </Flex>
                                                </Flex>
                                            </Link>
                                        ),
                                        sections: [
                                            {
                                                id: 'ss',
                                                header: '',
                                                content: (item) => <></>,
                                            },
                                            {
                                                id: 'description',
                                                header: 'Description',
                                                content: (item) => (
                                                    <div className=" text-wrap">
                                                        {item.description}
                                                    </div>
                                                ),
                                            },
                                        ],
                                    }}
                                    items={data.map((item) => {
                                        return {
                                            id: item.id,
                                            name: item.name,
                                            description: item.description,
                                            icon: item.icon,
                                            page: item.page,
                                            label: item.label,
                                        }
                                    })}
                                />
                            </div>
                        </main>
                    </div>
                </div>
            </Flex>
            <Modal open={open} onClose={() => setOpen(false)}>
                <Title className="text-black !text-xl font-bold w-full text-center mb-4">
                    Create custom dashboards for teams, functions, and
                    roles—In-App or through BI tools like PowerBI, Tableau,
                    Looker, or Grafana.
                </Title>
                <Cal
                    namespace="try-enterprise"
                    calLink="team/opengovernance/try-enterprise"
                    style={{
                        width: '100%',
                        height: '100%',
                        overflow: 'scroll',
                    }}
                    config={{ layout: 'month_view' }}
                />
            </Modal> */}
            <Title className="text-black !text-xl font-bold w-full text-center mb-4">
                Create custom dashboards for teams, functions, and roles—In-App
                or through BI tools like PowerBI, Tableau, Looker, or Grafana.
            </Title>
            <Cal
                namespace="try-enterprise"
                calLink="team/opengovernance/try-enterprise"
                style={{
                    width: '100%',
                    height: '100%',
                    overflow: 'scroll',
                }}
                config={{ layout: 'month_view' }}
            />
        </>
    )
}
