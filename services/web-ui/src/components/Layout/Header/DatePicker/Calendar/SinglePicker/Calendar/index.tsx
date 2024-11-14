import { useCalendarState } from 'react-stately'
import { useCalendar, useLocale } from 'react-aria'
import { createCalendar } from '@internationalized/date'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/react/24/solid'
import { CalendarButton } from '../../../Button'
import { CalendarGrid } from '../CalendarGrid'

export function Calendar(props: any) {
    const { locale } = useLocale()
    const state = useCalendarState({
        ...props,
        locale,
        createCalendar,
    })

    const { calendarProps, prevButtonProps, nextButtonProps, title } =
        useCalendar(props, state)

    return (
        <div {...calendarProps} className="inline-block">
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
