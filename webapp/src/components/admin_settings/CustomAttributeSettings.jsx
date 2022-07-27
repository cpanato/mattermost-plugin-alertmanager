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
        console.log('initial props');
        console.log(props.value);

        setSettings(initSettings(props.value));
    }, []);

    const initSettings = (newSettings) => {
        if(!!newSettings && newSettings.length > 0) {
            return new Map(newSettings.map((a, index) => [index, a]));
        }

        const emptySetting = [{ text: '' }];
        return new Map(emptySetting.map((a, index) => [index, a]));
    }

    const renderSettings = () => {
        if(settings.size === 0) {
            return (
                <div className='no-settings-alert'>{`No alert managers have been created`}</div>
            );
        }

        return Array.from(settings, ([key, value]) => {
            return (
                <AMAttribute
                    key={key}
                    id={key}
                    onChange={handleChange}
                    onDelete={triggerDeleteModal}
                    attributes = {{
                        teamName: value.teamName,
                        channelName: value.channelName,
                        token: value.token,
                        url: value.url,
                        error: value.error
                    }}
                />
            );
        });
    }

    const handleChange = ( { id, attributes } ) => {
        let newSettings = settings;
        newSettings.set(id, attributes);

        setSettings(newSettings);

        props.onChange(props.id, Array.from(newSettings.values()));
        props.setSaveNeeded();
    }

    const handleAddButtonClick = (e) => {
        e.preventDefault();

        const nextKey = [...settings.keys()].pop()+1;

        let newSettings = settings;
        newSettings.set(nextKey, {});

        setSettings(newSettings);

        props.onChange(props.id, Array.from(newSettings.values()));
        props.setSaveNeeded();
    }

    const handleDelete = (id) => {
        let newSettings = settings;
        newSettings.delete(id);

        setSettings(newSettings);
        setIsDeleteModalShown(false);

        props.onChange(props.id, Array.from(newSettings.values()));
        props.setSaveNeeded();
    }

    const triggerDeleteModal = (id) => {
        setIsDeleteModalShown(true);
        setSettingIdToDelete(id);
    };

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