"""Setup script for feedpulse"""

from setuptools import setup, find_packages

with open('requirements.txt') as f:
    requirements = f.read().splitlines()

setup(
    name='feedpulse',
    version='1.0.0',
    description='Concurrent Feed Aggregator CLI',
    author='AI Agent',
    packages=find_packages(),
    install_requires=requirements,
    entry_points={
        'console_scripts': [
            'feedpulse=feedpulse.cli:main',
        ],
    },
    python_requires='>=3.8',
    classifiers=[
        'Development Status :: 4 - Beta',
        'Intended Audience :: Developers',
        'Programming Language :: Python :: 3',
        'Programming Language :: Python :: 3.8',
        'Programming Language :: Python :: 3.9',
        'Programming Language :: Python :: 3.10',
        'Programming Language :: Python :: 3.11',
    ],
)
