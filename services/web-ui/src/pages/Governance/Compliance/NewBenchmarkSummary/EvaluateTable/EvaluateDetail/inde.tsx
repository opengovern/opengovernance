// @ts-nocheck
import { useAtomValue } from 'jotai'
import { ICellRendererParams, ValueFormatterParams } from 'ag-grid-community'
import { useCallback, useEffect, useMemo, useState } from 'react'
import {
    Button,
    Flex,
    MultiSelect,
    MultiSelectItem,
    Text,
    Title,
} from '@tremor/react'
import {
    ArrowPathRoundedSquareIcon,
    CloudIcon,
    PlayCircleIcon,
} from '@heroicons/react/24/outline'
import { Checkbox, useCheckboxState } from 'pretty-checkbox-react'
import { useComplianceApiV1AssignmentsBenchmarkDetail } from '../../../../../api/compliance.gen'
import {
    GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkAssignedConnection,
    GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkEvaluationSummary,
} from '../../../../../api/api'
import DrawerPanel from '../../../../../components/DrawerPanel'
import Table, { IColumn } from '../../../../../components/Table'
import { isDemoAtom } from '../../../../../store'
import KFilter from '../../../../../components/Filter'
import {
    Box,
    DateRangePicker,
    Icon,
    SpaceBetween,
    Spinner,
} from '@cloudscape-design/components'

import KMulstiSelect from '@cloudscape-design/components/multiselect'
import { Fragment, ReactNode } from 'react'
import { Dialog, Transition } from '@headlessui/react'
import Modal from '@cloudscape-design/components/modal'
import KButton from '@cloudscape-design/components/button'
import axios from 'axios'
import KTable from '@cloudscape-design/components/table'
import KeyValuePairs from '@cloudscape-design/components/key-value-pairs'
import Badge from '@cloudscape-design/components/badge'
import {
    BreadcrumbGroup,
    Header,
    Link,
    Pagination,
    PropertyFilter,
} from '@cloudscape-design/components'
import { AppLayout, SplitPanel } from '@cloudscape-design/components'
import { dateTimeDisplay } from '../../../../../utilities/dateDisplay'
import StatusIndicator from '@cloudscape-design/components/status-indicator'
import SeverityBar from '../../BenchmarkCard/SeverityBar'

const JOB_STATUS = {
    CANCELED: 'stopped',
    SUCCEEDED: '',
    FAILED: 'error',
    SUMMARIZER_IN_PROGRESS: 'in-progress',
    SINK_IN_PROGRESS: 'in-progress',
    RUNNERS_IN_PROGRESS: 'in-progress',
}

export default function EvaluateDetail() {
   
   
   

    const GetDetail = () => {
        // /compliance/api/v3/benchmark/{benchmark-id}/assignments
        setDetailLoading(true)
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
        let connector = ''
        benchmarkDetail?.connectors?.map((c) => {
            connector += `connector=${c}&`
        })
        axios
            .get(
                // @ts-ignore
                `${url}/main//compliance/api/v3/compliance/summary/${selected.job_id} `,
                config
            )
            .then((res) => {
                //   setAccounts(res.data.integrations)
                setDetail(res.data)
                setDetailLoading(false)
            })
            .catch((err) => {
                setDetailLoading(false)
                console.log(err)
            })
    }
    useEffect(() => {
        GetHistory()
    }, [page, jobStatus, date, selectedIntegrations])

    useEffect(() => {
        GetIntegrations()
    }, [])

    useEffect(() => {
        if (selected) {
            GetDetail()
        }
    }, [selected])
    const truncate = (text: string | undefined) => {
        if (text) {
            return text.length > 30 ? text.substring(0, 30) + '...' : text
        }
    }
    return (
        <>
            
        </>
    )
}
