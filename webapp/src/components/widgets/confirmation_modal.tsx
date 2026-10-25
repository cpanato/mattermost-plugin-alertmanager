import React, {useEffect, useRef} from 'react';

type Props = {
    show: boolean;
    title: React.ReactNode;
    message: React.ReactNode;
    confirmButtonText: React.ReactNode;
    onConfirm: () => void;
    onCancel: () => void;
};

// A native dialog avoids the legacy React APIs in the old Bootstrap modal.
export default function ConfirmModal(props: Props) {
    const dialog = useRef<HTMLDialogElement>(null);
    useEffect(() => {
        const element = dialog.current;
        if (!element) {
            return;
        }
        if (props.show && !element.open) {
            element.showModal();
        } else if (!props.show && element.open) {
            element.close();
        }
        return () => element.close();
    }, [props.show]);

    return (
        <dialog
            ref={dialog}
            className='alertmanager-confirm-dialog'
            aria-labelledby='alertmanager-confirm-title'
            onCancel={(event) => {
                event.preventDefault();
                props.onCancel();
            }}
        >
            <h2 id='alertmanager-confirm-title'>{props.title}</h2>
            <p>{props.message}</p>
            <button type='button' className='btn btn-default' autoFocus={true} onClick={props.onCancel}>
                {'Cancel'}
            </button>
            <button type='button' className='btn btn-primary' onClick={props.onConfirm}>
                {props.confirmButtonText}
            </button>
        </dialog>
    );
}
