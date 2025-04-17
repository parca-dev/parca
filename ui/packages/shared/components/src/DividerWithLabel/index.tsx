import cx from 'classnames'

export const DividerWithLabel = ({ label, className = '' }: { label: string, className?: string }) => {
    return (<div className={cx("relative", className)}>
        <div aria-hidden="true" className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-gray-300" />
        </div>
        <div className="relative flex justify-start">
            <span className="bg-white pr-2 text-xs text-gray-500 uppercase">{label}</span>
        </div>
    </div>);
}
