// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import PropTypes from 'prop-types';
import React, { useState, useEffect } from 'react';

import AMAttribute from './AMAttribute';
import ConfirmModal from '../widgets/confirmation_modal';
import '../../styles/main.css';

const CustomAttributesSettings = (props) => {
    const [ settings, setSettings ] = useState(new Map());
    const [ isDeleteModalShown, setIsDeleteModalShown ] = useState(false);
    const [ settingIdToDelete, setSettingIdToDelete ] = useState();

    useEffect(() => {
        setSettings(initSettings(props.value));
    },[]);

    const initSettings = (newSettings) => {
        if(!!newSettings) {
            if(Object.keys(newSettings).length != 0) {
                const newEntries = Object.entries(newSettings);

                return new Map(newEntries);
            }
        }

        const emptySetting = { '0' : {
                alertmanagerurl: '',
                channel: '',
                team: '',
                token: ''
            }
        };

        return new Map(Object.entries(emptySetting));
    }

    const handleChange = ( { id, attributes } ) => {
        let newSettings = settings;
        newSettings.set(id, attributes);

        setSettings(newSettings);

        props.onChange(props.id, Object.fromEntries(newSettings));
        props.setSaveNeeded();
    }

    const handleAddButtonClick = (e) => {
        e.preventDefault();

        const nextKey = settings.size === 0 ? '0' :(parseInt([...settings.keys()].pop()) + 1).toString();

        let newSettings = settings;
        newSettings.set(nextKey, {});

        setSettings(newSettings);

        props.onChange(props.id, Object.fromEntries(newSettings));
        props.setSaveNeeded();
    }

    const handleDelete = (id) => {
        let newSettings = settings;
        newSettings.delete(id);

        setSettings(newSettings);
        setIsDeleteModalShown(false);

        props.onChange(props.id, Object.fromEntries(newSettings));
        props.setSaveNeeded();
    }

    const triggerDeleteModal = (id) => {
        setIsDeleteModalShown(true);
        setSettingIdToDelete(id);
    };

    const renderSettings = () => {
        if(settings.size === 0) {
            return (
                <div className='no-settings-alert'>{`No alert managers have been created`}</div>
            );
        }

        return Array.from(settings, ([key, value], index) => {
            return (
                <AMAttribute
                    key={key}
                    id={key}
                    orderNumber={index}
                    onChange={handleChange}
                    onDelete={triggerDeleteModal}
                    attributes = {{
                        team: value.team,
                        channel: value.channel,
                        token: value.token,
                        alertmanagerurl: value.alertmanagerurl
                    }}
                />
            );
        });
    }

    return (
        <div className='alert-setting__wrapper'>
            {renderSettings()}
            <div className='alert-setting__add-wrapper'>
                <button
                    className='alert-setting__add-button btn btn-primary'
                    onClick={handleAddButtonClick}
                >
                    {`Add new alert manager`}
                </button>
            </div>
            <ConfirmModal
                    show={isDeleteModalShown}
                    title={'Delete Alert Manager'}
                    message={
                        'Are you sure you want to remove this alert manager?'
                    }
                    confirmButtonText={'Remove'}
                    onConfirm={() => {
                        handleDelete(settingIdToDelete);
                    }}
                    onCancel={() => setIsDeleteModalShown(false)}
                />
        </div>
    );
}

CustomAttributesSettings.propTypes =  {
    id: PropTypes.string.isRequired,
    label: PropTypes.string.isRequired,
    helpText: PropTypes.node,
    value: PropTypes.any,
    disabled: PropTypes.bool.isRequired,
    config: PropTypes.object.isRequired,
    currentState: PropTypes.object,
    license: PropTypes.object.isRequired,
    setByEnv: PropTypes.bool.isRequired,
    onChange: PropTypes.func.isRequired,
    registerSaveAction: PropTypes.func.isRequired,
    setSaveNeeded: PropTypes.func.isRequired,
    unRegisterSaveAction: PropTypes.func.isRequired,
}

export default CustomAttributesSettings;