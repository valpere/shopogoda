#!/usr/bin/env perl

=head1 NAME

convert_locales.pl - Convert locale files between CSV and JSON formats

=head1 SYNOPSIS

    perl convert_locales.pl [OPTIONS]

=head1 DESCRIPTION

This script converts ShoPogoda bot locale files between CSV and JSON formats.
It processes all locale files in batch.

Directories:
- CSV:  $PROJECT_ROOT/locales/
- JSON: $PROJECT_ROOT/internal/locales/

=head1 OPTIONS

    --csv-to-json    Convert CSV files to JSON (default)
    --json-to-csv    Convert JSON files to CSV
    --help          Show this help message
    --verbose       Enable verbose output

=head1 EXAMPLES

    # Convert CSV to JSON (default)
    perl convert_locales.pl

    # Convert JSON to CSV
    perl convert_locales.pl --json-to-csv

    # Convert with verbose output
    perl convert_locales.pl --json-to-csv --verbose

=cut

use strict;
use warnings;
use utf8;
use Getopt::Long;
use Pod::Usage;
use JSON::PP;
use Text::CSV;
use File::Spec;
use File::Basename;
use Cwd 'abs_path';

# Enable UTF-8 for all I/O
binmode(STDOUT, ':encoding(UTF-8)');
binmode(STDERR, ':encoding(UTF-8)');

# Configuration
my $help = 0;
my $verbose = 0;
my $csv_to_json = 1;  # Default direction
my $json_to_csv = 0;

# Parse command line options
GetOptions(
    'csv-to-json'   => \$csv_to_json,
    'json-to-csv'   => \$json_to_csv,
    'help|h'        => \$help,
    'verbose|v'     => \$verbose,
) or pod2usage(2);

pod2usage(1) if $help;

# Determine conversion direction
if ($json_to_csv) {
    $csv_to_json = 0;
}

# Find project root (directory containing this script's parent)
my $script_dir = dirname(abs_path($0));
my $project_root = dirname($script_dir);

# Define directories
my $csv_dir = File::Spec->catdir($project_root, 'locales');
my $json_dir = File::Spec->catdir($project_root, 'internal', 'locales');

print "Project root: $project_root\n" if $verbose;
print "CSV directory: $csv_dir\n" if $verbose;
print "JSON directory: $json_dir\n" if $verbose;

# Create directories if they don't exist
unless (-d $csv_dir) {
    mkdir $csv_dir or die "Cannot create CSV directory '$csv_dir': $!";
    print "Created CSV directory: $csv_dir\n" if $verbose;
}

unless (-d $json_dir) {
    mkdir $json_dir or die "Cannot create JSON directory '$json_dir': $!";
    print "Created JSON directory: $json_dir\n" if $verbose;
}

# Locale file patterns
my @locales = qw(en de es fr uk);

if ($csv_to_json) {
    print "Converting CSV to JSON...\n";
    convert_csv_to_json();
} else {
    print "Converting JSON to CSV...\n";
    convert_json_to_csv();
}

print "Conversion complete!\n";

=head1 SUBROUTINES

=head2 convert_csv_to_json

Converts all CSV locale files to JSON format.

=cut

sub convert_csv_to_json {
    for my $locale (@locales) {
        my $csv_file = File::Spec->catfile($csv_dir, "$locale.csv");
        my $json_file = File::Spec->catfile($json_dir, "$locale.json");

        unless (-f $csv_file) {
            print "Warning: CSV file not found: $csv_file\n";
            next;
        }

        print "Converting $csv_file -> $json_file\n" if $verbose;

        # Read CSV file
        my $csv = Text::CSV->new({
            binary => 1,
            auto_diag => 1,
            sep_char => ',',
        });

        open my $csv_fh, '<:encoding(utf8)', $csv_file
            or die "Cannot open CSV file '$csv_file': $!";

        # Skip header row
        my $header = $csv->getline($csv_fh);
        unless ($header && @$header >= 2) {
            die "Invalid CSV header in '$csv_file'. Expected: Key,Value";
        }

        # Read all translations
        my %translations;
        while (my $row = $csv->getline($csv_fh)) {
            next unless @$row >= 2;
            my ($key, $value) = @$row;

            # Skip empty keys
            next unless defined $key && $key ne '';

            # Handle undefined values
            $value = '' unless defined $value;

            $translations{$key} = $value;
        }

        close $csv_fh;

        # Write JSON file
        my $json = JSON::PP->new->pretty->canonical;

        open my $json_fh, '>:encoding(utf8)', $json_file
            or die "Cannot create JSON file '$json_file': $!";

        print $json_fh $json->encode(\%translations);
        close $json_fh;

        my $count = keys %translations;
        print "  -> Converted $count translations\n" if $verbose;
    }
}

=head2 convert_json_to_csv

Converts all JSON locale files to CSV format.

=cut

sub convert_json_to_csv {
    for my $locale (@locales) {
        my $json_file = File::Spec->catfile($json_dir, "$locale.json");
        my $csv_file = File::Spec->catfile($csv_dir, "$locale.csv");

        unless (-f $json_file) {
            print "Warning: JSON file not found: $json_file\n";
            next;
        }

        print "Converting $json_file -> $csv_file\n" if $verbose;

        # Read JSON file
        open my $json_fh, '<:encoding(utf8)', $json_file
            or die "Cannot open JSON file '$json_file': $!";

        my $json_content = do { local $/; <$json_fh> };
        close $json_fh;

        my $json = JSON::PP->new;
        my $translations = $json->decode($json_content);

        unless (ref $translations eq 'HASH') {
            die "Invalid JSON format in '$json_file'. Expected hash object.";
        }

        # Write CSV file
        my $csv = Text::CSV->new({
            binary => 1,
            auto_diag => 1,
            sep_char => ',',
            eol => "\n",
            quote_space => 0,
        });

        open my $csv_fh, '>:encoding(utf8)', $csv_file
            or die "Cannot create CSV file '$csv_file': $!";

        # Write header
        $csv->print($csv_fh, ['Key', 'Value']);

        # Write translations (sorted by key for consistency)
        for my $key (sort keys %$translations) {
            my $value = $translations->{$key};
            $value = '' unless defined $value;
            $csv->print($csv_fh, [$key, $value]);
        }

        close $csv_fh;

        my $count = keys %$translations;
        print "  -> Converted $count translations\n" if $verbose;
    }
}

=head1 AUTHOR

ShoPogoda Bot Development Team

=head1 LICENSE

This script is part of the ShoPogoda project.

=cut