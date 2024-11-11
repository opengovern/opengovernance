import { useRef } from 'react'
import { useRangeCalendarState } from 'react-stately'
import { useRangeCalendar, useLocale } from 'react-aria'
import { createCalendar } from '@internationalized/date'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/react/24/solid'
import { CalendarButton } from '../../../Button'
import { CalendarGrid } from '../CalendarGrid'

export function RangeCalendar(props: any) {
    const { locale } = useLocale()
    const state = useRangeCalendarState({
        ...props,
        locale,
        createCalendar,
    })

    const ref = useRef(null)
    const { calendarProps, prevButtonProps, nextButtonProps, title } =
        useRangeCalendar(props, state, ref)

    return (
        <div {...calendarProps} ref={ref} className="inline-block">
            <div className="flex items-center pb-4">
                <h2 className="flex-1 font-bold text-xl ml-2">{title}</h2>
                <CalendarButton {...prevButtonProps}>
                    <ChevronLeftIcon className="h-6 w-6" />
                </CalendarButton>
                <CalendarButton {...nextButtonProps}>
                    <ChevronRightIcon className="h-6 w-6" />
                </CalendarButton>
            </div>
            <CalendarGrid state={state} />
        </div>
    )
}
